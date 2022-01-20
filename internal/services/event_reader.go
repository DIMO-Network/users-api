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
	"github.com/rs/zerolog"
	"github.com/volatiletech/sqlboiler/v4/boil"
	"github.com/volatiletech/sqlboiler/v4/queries/qm"
)

const integrationCreationEventType = "com.dimo.zone.device.integration.create"
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

// processEvent waits for a user device integration creation event and creates a referral if
// appropriate.
func (e *EventReader) processEvent(msg *message.Message) error {
	ctx := context.Background()
	// Keep the pipeline moving, deal with the fallout later.
	defer func() { msg.Ack() }()

	var msgParts struct {
		Type string          `json:"type"`
		Data json.RawMessage `json:"data"`
	}
	err := json.Unmarshal(msg.Payload, &msgParts)
	if err != nil {
		return fmt.Errorf("could not parse event: %w", err)
	}

	if msgParts.Type != integrationCreationEventType {
		return nil
	}

	var data integrationCreationData
	err = json.Unmarshal(msgParts.Data, &data)
	if err != nil {
		return fmt.Errorf("could not parse integration creation event data: %w", err)
	}

	if len(data.Device.VIN) != 17 {
		return fmt.Errorf("received integration creation event with invalid VIN %s", data.Device.VIN)
	}

	tx, err := e.db().Writer.BeginTx(context.Background(), nil)
	if err != nil {
		return fmt.Errorf("failed to create transaction: %w", err)
	}
	defer tx.Rollback() //nolint

	// The referred user may be deleted later, but at this time he should exist.
	referredUser, err := models.FindUser(ctx, tx, data.UserID)
	if err != nil {
		return fmt.Errorf("failed to find user with id %s who created integration: %w", data.UserID, err)
	}

	// If no referrer, then there is nothing to do.
	if !referredUser.ReferrerID.Valid {
		return nil
	}

	// See if this user or vehicle has already been used for a referral.
	conflictingReferral, err := models.Referrals(
		models.ReferralWhere.ReferredUserID.EQ(data.UserID),
		qm.Or2(models.ReferralWhere.Vin.EQ(data.Device.VIN)),
	).One(ctx, tx)
	if err == nil {
		if conflictingReferral.Vin == data.Device.VIN {
			e.log.Info().Msgf("VIN %s has already been used in a referral", data.Device.VIN)
		} else {
			e.log.Info().Msgf("User %s has already completed a referral", data.UserID)
		}
		return nil
	} else if !errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("failed searching for conflicting referrals: %w", err)
	}

	// Should be able to create the referral in the database without issue.
	referral := models.Referral{
		UserID:         referredUser.ReferrerID.String,
		ReferredUserID: data.UserID,
		Vin:            data.Device.VIN,
	}
	err = referral.Insert(context.Background(), tx, boil.Infer())
	if err != nil {
		return fmt.Errorf("failed to insert referral record: %w", err)
	}

	err = tx.Commit()
	if err != nil {
		return fmt.Errorf("failed to commit new referral: %w", err)
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

	return nil
}
