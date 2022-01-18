package services

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/DIMO-INC/users-api/internal/database"
	"github.com/DIMO-INC/users-api/models"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/lib/pq"
	"github.com/rs/zerolog"
	"github.com/volatiletech/sqlboiler/v4/boil"
	"github.com/volatiletech/sqlboiler/v4/queries/qm"
)

const referralCompleteEventType = "com.dimo.zone.user.referral.complete"

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
		VIN string `json:"vin"`
	} `json:"device"`
}

type referralEventData struct {
	Timestamp time.Time `json:"timestamp"`
	UserID    string    `json:"userId"`
}

func (e *EventReader) processEvent(msg *message.Message) error {
	ack := true
	defer func() {
		if ack {
			msg.Ack()
		}
	}()
	var msgTypeOnly struct {
		Type string          `json:"type"`
		Data json.RawMessage `json:"data"`
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

	if data.Device.VIN == "" {
		return nil
	}

	tx, err := e.db().Writer.BeginTx(context.Background(), nil)
	if err != nil {
		ack = false
		return err
	}
	defer tx.Rollback() //nolint

	// Find the user who registered the integration, then find out if he was referred by someone.
	user, err := models.Users(
		models.UserWhere.ID.EQ(data.UserID),
		qm.Load(models.UserRels.Referrer),
	).One(context.Background(), tx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("Got an integration event for a user %s we don't have", data.UserID)
		}
		ack = false
		return err
	}

	referrer := user.R.Referrer

	if referrer == nil {
		return nil
	}

	referral := models.Referral{
		UserID:         referrer.ID,
		ReferredUserID: data.UserID,
		Vin:            data.Device.VIN,
	}

	err = referral.Insert(context.Background(), tx, boil.Infer())
	if err != nil {
		var pqErr *pq.Error
		if !errors.As(err, &pqErr) || pqErr.Code != "23505" {
			ack = false
			return err
		}
		// Some errors are entirely expected here.
		return err
	}

	err = e.eventService.Emit(&Event{
		Type:    referralCompleteEventType,
		Subject: data.UserID,
		Source:  "users-api",
		Data: referralEventData{
			Timestamp: time.Now(),
			UserID:    data.UserID,
		},
	})
	if err != nil {
		e.log.Err(err).Msg("Failed to send referral event")
	}

	err = tx.Commit()
	if err != nil {
		return err
	}

	return nil
}
