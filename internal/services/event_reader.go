package services

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/DIMO-INC/users-api/internal/database"
	"github.com/DIMO-INC/users-api/models"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/rs/zerolog"
	"github.com/volatiletech/sqlboiler/v4/boil"
	"github.com/volatiletech/sqlboiler/v4/queries/qm"
)

type EventReader struct {
	db           func() *database.DBReaderWriter
	log          *zerolog.Logger
	eventService *EventService
}

func NewEventReader(db func() *database.DBReaderWriter, logger *zerolog.Logger, eventService *EventService) *EventReader {
	return &EventReader{
		db:           db,
		log:          logger,
		eventService: eventService,
	}
}

func (e *EventReader) ProcessDeviceStatusMessages(messages <-chan *message.Message) {
	for msg := range messages {
		err := e.processEvent(msg)
		if err != nil {
			e.log.Err(err).Msg("error processing event")
		}
	}
}

type integrationCreationData struct {
	UserID string `json:"userId"`
	Device struct {
		VIN *string `json:"vin"`
	} `json:"device"`
}

func (e *EventReader) processEvent(msg *message.Message) error {
	var msgTypeOnly struct {
		Type string          `json:"type"`
		Data json.RawMessage `json:data`
	}
	err := json.Unmarshal(msg.Payload, &msgTypeOnly)
	if err != nil {
		return fmt.Errorf("could not get type from event message: %w", err)
	}

	if msgTypeOnly.Type != "com.dimo.zone.device.integration.create" {
		return nil
	}

	var data integrationCreationData
	err = json.Unmarshal(msgTypeOnly.Data, &data)
	if err != nil {
		return fmt.Errorf("could not parse integration creation data: %w", err)
	}

	tx, err := e.db().Writer.BeginTx(context.Background(), nil)
	if err != nil {
		return err
	}

	user, err := models.Users(
		models.UserWhere.ID.EQ(data.UserID),
		qm.Load(models.UserRels.Referrer),
	).One(context.Background(), tx)
	if err != nil {
		return err
	}

	referrer := user.R.Referrer

	if referrer == nil {
		return nil
	}

	referral := models.Referral{
		UserID:         referrer.ID,
		ReferredUserID: data.UserID,
		Vin:            *data.Device.VIN,
	}

	err = referral.Insert(context.Background(), tx, boil.Infer())
	if err != nil {
		// Some errors are entirely expected here.
		return err
	}

	return nil
}
