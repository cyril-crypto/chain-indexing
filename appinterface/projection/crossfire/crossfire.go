package crossfire

import (
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"

	"github.com/crypto-com/chain-indexing/appinterface/projection/crossfire/constants"
	"github.com/crypto-com/chain-indexing/appinterface/projection/crossfire/view"
	"github.com/crypto-com/chain-indexing/appinterface/projection/rdbprojectionbase"
	"github.com/crypto-com/chain-indexing/appinterface/rdb"
	event_entity "github.com/crypto-com/chain-indexing/entity/event"
	applogger "github.com/crypto-com/chain-indexing/internal/logger"
	"github.com/crypto-com/chain-indexing/internal/tmcosmosutils"
	"github.com/crypto-com/chain-indexing/internal/utctime"
	event_usecase "github.com/crypto-com/chain-indexing/usecase/event"
)

type Crossfire struct {
	*rdbprojectionbase.Base

	conNodeAddressPrefix     string
	validatorAddressPrefix   string
	phaseOneStartTime        utctime.UTCTime
	phaseTwoStartTime        utctime.UTCTime
	phaseThreeStartTime      utctime.UTCTime
	competitionEndTime       utctime.UTCTime
	adminAddress             string
	networkUpgradeProposalID string
	rdbConn                  rdb.Conn
	logger                   applogger.Logger
}

func NewCrossfire(
	logger applogger.Logger,
	rdbConn rdb.Conn,
	conNodeAddressPrefix string,
	validatorAddressPrefix string,
	unixPhaseOneStartTime int64,
	unixPhaseTwoStartTime int64,
	unixPhaseThreeStartTime int64,
	unixCompetitionEndTime int64,
	adminAddress string,
	networkUpgradeProposalID string,
) *Crossfire {
	return &Crossfire{
		Base: rdbprojectionbase.NewRDbBase(rdbConn.ToHandle(), "Crossfire"),

		conNodeAddressPrefix:     conNodeAddressPrefix,
		validatorAddressPrefix:   validatorAddressPrefix,
		phaseOneStartTime:        utctime.FromUnixNano(unixPhaseOneStartTime),
		phaseTwoStartTime:        utctime.FromUnixNano(unixPhaseTwoStartTime),
		phaseThreeStartTime:      utctime.FromUnixNano(unixPhaseThreeStartTime),
		competitionEndTime:       utctime.FromUnixNano(unixCompetitionEndTime),
		adminAddress:             adminAddress, // TODO: address prefix check
		networkUpgradeProposalID: networkUpgradeProposalID,
		rdbConn:                  rdbConn,
		logger:                   logger,
	}
}

func (_ *Crossfire) GetEventsToListen() []string {
	return []string{
		event_usecase.BLOCK_CREATED,
		event_usecase.MSG_CREATE_VALIDATOR_CREATED,
		event_usecase.MSG_VOTE_CREATED,
		event_usecase.MSG_SUBMIT_SOFTWARE_UPGRADE_PROPOSAL_CREATED,
	}
}

func (projection *Crossfire) OnInit() error {
	return nil
}

func (projection *Crossfire) HandleEvents(height int64, events []event_entity.Event) error {
	rdbTx, err := projection.rdbConn.Begin()
	if err != nil {
		return fmt.Errorf("error beginning transaction: %v", err)
	}

	committed := false
	defer func() {
		if !committed {
			_ = rdbTx.Rollback()
		}
	}()

	rdbTxHandle := rdbTx.ToHandle()
	crossfireValidatorsView := view.NewCrossfireValidators(rdbTxHandle)
	crossfireChainStatsView := view.NewCrossfireChainStats(rdbTxHandle)
	crossfireValidatorStatsView := view.NewCrossfireValidatorsStats(rdbTxHandle)
	var blockTime utctime.UTCTime
	var blockHash string
	for _, event := range events {
		if blockCreatedEvent, ok := event.(*event_usecase.BlockCreated); ok {
			blockTime = blockCreatedEvent.Block.Time
			blockHash = blockCreatedEvent.Block.Hash
		}
	}
	// TODO: remove print for projectValidatorTx view
	fmt.Println(blockTime, blockHash)

	// TODO: views preparation starts
	if err := projection.projectCrossfireValidatorView(crossfireValidatorsView, crossfireChainStatsView, crossfireValidatorStatsView, height, blockTime, events); err != nil {
		return fmt.Errorf("error projecting validator view: %v", err)
	}
	// TODO ends: views preparation ends and update current height as handled

	if err := projection.UpdateLastHandledEventHeight(rdbTxHandle, height); err != nil {
		return fmt.Errorf("error updating last handled event height: %v", err)
	}

	if err := rdbTx.Commit(); err != nil {
		return fmt.Errorf("error committing changes: %v", err)
	}
	committed = true
	return nil
}

func (projection *Crossfire) projectCrossfireValidatorView(
	crossfireValidatorsView *view.CrossfireValidators,
	crossfireChainStatsView *view.CrossfireChainStats,
	crossfireValidatorStatsView *view.CrossfireValidatorsStats,
	blockHeight int64,
	blockTime utctime.UTCTime,
	events []event_entity.Event,
) error {
	// MsgCreateValidator should be handled first
	for _, event := range events {
		if msgCreateValidatorEvent, ok := event.(*event_usecase.MsgCreateValidator); ok {
			projection.logger.Debug("handling MsgCreateValidator event")
			pubKey, err := base64.StdEncoding.DecodeString(msgCreateValidatorEvent.TendermintPubkey)
			if err != nil {
				return fmt.Errorf("error base64 decoding Tendermint node pubkey: %v", err)
			}
			consensusNodeAddress, err := tmcosmosutils.ConsensusNodeAddressFromTmPubKey(
				projection.conNodeAddressPrefix, pubKey,
			)
			if err != nil {
				return fmt.Errorf("error converting consensus node pubkey to address: %v", err)
			}
			validatorRow := view.CrossfireValidatorRow{
				ConsensusNodeAddress:            consensusNodeAddress,
				OperatorAddress:                 msgCreateValidatorEvent.ValidatorAddress,
				InitialDelegatorAddress:         msgCreateValidatorEvent.DelegatorAddress,
				Status:                          constants.UNBONDED,
				Jailed:                          false,
				JoinedAtBlockHeight:             blockHeight,
				JoinedAtBlockTime:               blockTime,
				Moniker:                         msgCreateValidatorEvent.Description.Moniker,
				Identity:                        msgCreateValidatorEvent.Description.Identity,
				Website:                         msgCreateValidatorEvent.Description.Website,
				SecurityContact:                 msgCreateValidatorEvent.Description.SecurityContact,
				Details:                         msgCreateValidatorEvent.Description.Details,
				TaskPhase1NodeSetup:             constants.INCOMPLETED,
				TaskPhase2KeepNodeActive:        constants.INCOMPLETED,
				TaskPhase2ProposalVote:          constants.INCOMPLETED,
				TaskPhase2NetworkUpgrade:        constants.INCOMPLETED,
				RankTaskPhase1n2CommitmentCount: 0,
				RankTaskPhase3CommitmentCount:   0,
				RankTaskHighestTxSent:           0,
			}

			isJoined, joinedAtBlockHeight, joinedAtBlockTime, err := crossfireValidatorsView.LastJoinedBlockHeight(
				validatorRow.OperatorAddress, validatorRow.ConsensusNodeAddress,
			)
			if err != nil {
				return fmt.Errorf("error querying validator last joined block height: %v", err)
			}
			if isJoined {
				validatorRow.JoinedAtBlockHeight = joinedAtBlockHeight
				validatorRow.JoinedAtBlockTime = joinedAtBlockTime
			}

			if err := crossfireValidatorsView.Upsert(&validatorRow); err != nil {
				return fmt.Errorf("error inserting new validator into view: %v", err)
			}

			// checkTaskSetup
			if err := projection.checkTaskSetup(
				crossfireValidatorsView,
				validatorRow.OperatorAddress,
				validatorRow.ConsensusNodeAddress,
				validatorRow.JoinedAtBlockTime,
			); err != nil {
				return fmt.Errorf("error check Setup task for new validator: %v", err)
			}
		}
	}
	for _, event := range events {
		if msgSubmitSoftwareUpgradeProposalEvent, ok := event.(*event_usecase.MsgSubmitSoftwareUpgradeProposal); ok {
			projection.logger.Debug("handling MsgSubmitSoftwareUpgradeProposal event")

			// Check if proposed after competition has ended
			if blockTime.After(projection.competitionEndTime) {
				return fmt.Errorf("error Competition has already ended")
			}

			// Check if proposed before OR after Phase 2
			if blockTime.Before(projection.phaseTwoStartTime) || blockTime.After(projection.phaseThreeStartTime) {
				return fmt.Errorf("error This proposal does not occur in Phase 2")
			}

			// Check if proposed by NOT an admin
			if msgSubmitSoftwareUpgradeProposalEvent.ProposerAddress != projection.adminAddress {
				return fmt.Errorf("error checking proposer address equals admin address")
			}

			// Check if proposal ID does not match the required ID
			if msgSubmitSoftwareUpgradeProposalEvent.MaybeProposalId != nil && *msgSubmitSoftwareUpgradeProposalEvent.MaybeProposalId != projection.networkUpgradeProposalID {
				return fmt.Errorf("error checking Proposal ID in proposal")
			}

			networkUpgradeTimestamp := msgSubmitSoftwareUpgradeProposalEvent.MsgSubmitSoftwareUpgradeProposalParams.Content.Plan.Time
			networkUpgradeBlockheight := msgSubmitSoftwareUpgradeProposalEvent.MsgSubmitSoftwareUpgradeProposalParams.Content.Plan.Height

			// Update Network upgrade target Timestamp in DB
			network_upgrade_timestamp_dbkey := constants.NETWORK_UPGRADE + constants.DB_KEY_SEPARATOR + "timestamp"
			errTSUpdate := crossfireChainStatsView.Set(network_upgrade_timestamp_dbkey, networkUpgradeTimestamp.UnixNano())
			if errTSUpdate != nil {
				return fmt.Errorf("error updating network_upgrade timestamp: %v", errTSUpdate)
			}

			// Update Network upgrade target Blockheight in DB
			network_upgrade_blockheight_dbkey := constants.NETWORK_UPGRADE + constants.DB_KEY_SEPARATOR + "blockheight"
			errBlockheightUpdate := crossfireChainStatsView.Set(network_upgrade_blockheight_dbkey, networkUpgradeBlockheight)

			if errBlockheightUpdate != nil {
				return fmt.Errorf("error updating network_upgrade blockheight: %v", errBlockheightUpdate)
			}

		} else if msgVoteCreated, ok := event.(*event_usecase.MsgVote); ok {
			projection.logger.Debug("handling MsgVote event")

			// Check if proposed after competition has ended
			if blockTime.After(projection.competitionEndTime) {
				return fmt.Errorf("error Competition has already ended")
			}

			// Check if proposed before OR after Phase 2
			if blockTime.Before(projection.phaseTwoStartTime) || blockTime.After(projection.phaseThreeStartTime) {
				return fmt.Errorf("error Ineligible vote as it does not occur in Phase 2")
			}

			// Check if proposal ID does not match the required ID
			if msgVoteCreated.ProposalId != "" && msgVoteCreated.ProposalId != projection.networkUpgradeProposalID {
				return fmt.Errorf("error checking Proposal ID in Vote not matching")
			}

			// Check if Vote is NOT Yes or Abstain
			// TODO: Whether keep VOTE_OPTION_UNSPECIFIED or not?
			if !(strings.ToUpper(msgVoteCreated.Option) == constants.VOTE_OPTION_YES || strings.ToUpper(msgVoteCreated.Option) == constants.VOTE_OPTION_ABSTAIN || strings.ToUpper(msgVoteCreated.Option) == constants.VOTE_OPTION_UNSPECIFIED) {
				return fmt.Errorf("error Ineligible Vote. Casted vote: %s", msgVoteCreated.Option)
			}

			//TODO: Is this assumption correct? How to fetch operatorAddress / consensusAddress
			operatorAddress, errConverting := tmcosmosutils.ValidatorAddressFromPubAddress(projection.validatorAddressPrefix, msgVoteCreated.Voter)

			if errConverting != nil {
				return fmt.Errorf("error In converting voter address to validator address: s%v", errConverting)
			}

			errCheckingTask := projection.checkTaskNetworkProposalVote(crossfireValidatorsView, operatorAddress, blockTime)

			if errCheckingTask != nil {
				return fmt.Errorf("error In checking Network proposal vote: s%v", errCheckingTask)
			}
			// Update the proposed ID against the voter in Database
			voted_proposal_id_db_key := constants.VOTED_PROPOSAL_ID + constants.DB_KEY_SEPARATOR + msgVoteCreated.Voter

			proposalIdAsInt64, errConversion := strconv.ParseInt(msgVoteCreated.ProposalId, 10, 64)
			if errConversion != nil {
				return fmt.Errorf("error converting ProposalID to int64: s%v", errConversion)
			}
			errUpdateValidatorStats := crossfireValidatorStatsView.Set(voted_proposal_id_db_key, proposalIdAsInt64)

			if errUpdateValidatorStats != nil {
				return fmt.Errorf("error Updating ProposalID for the voter: s%v", errUpdateValidatorStats)
			}
		} else if msgTransactionCreated, ok := event.(*event_usecase.TransactionCreated); ok {
			projection.logger.Debug("handling TransactionCreated event")

			// Check if proposed after competition has ended
			if blockTime.After(projection.competitionEndTime) {
				return fmt.Errorf("error Competition has already ended")
			}

			// Update the Tx Count
			errUpdateTxCount := projection.updateTxSentCount(crossfireValidatorStatsView, blockTime, msgTransactionCreated)
			if errUpdateTxCount != nil {
				return fmt.Errorf("error Updating tx sent count: s%v", errUpdateTxCount)
			}
		}
	}
	return nil
}

// checkTaskSetup checks if node joins before phase 2 and update task_phase_1_node_setup
func (projection *Crossfire) checkTaskSetup(
	crossfireValidatorsView *view.CrossfireValidators,
	operatorAddress string,
	consensusNodeAddress string,
	joinedAtBlockTime utctime.UTCTime,
) error {
	if joinedAtBlockTime.Before(projection.phaseTwoStartTime) {
		if err := crossfireValidatorsView.UpdateTask(
			"task_phase_1_node_setup",
			constants.COMPLETED,
			operatorAddress,
			consensusNodeAddress,
		); err != nil {
			return fmt.Errorf("error updating validator TaskPhase1NodeSetup as completed: s%v", err)
		}
	}

	if joinedAtBlockTime.After(projection.phaseTwoStartTime) {
		if err := crossfireValidatorsView.UpdateTask(
			"task_phase_1_node_setup",
			constants.MISSED,
			operatorAddress,
			consensusNodeAddress,
		); err != nil {
			return fmt.Errorf("error updating validator TaskPhase1NodeSetup as missed: s%v", err)
		}
	}

	return nil
}

// checkTaskNetworkProposalVote
func (projection *Crossfire) checkTaskNetworkProposalVote(
	crossfireValidatorsView *view.CrossfireValidators,
	voterAddress string,
	txBlockTime utctime.UTCTime,
) error {
	if txBlockTime.Before(projection.phaseThreeStartTime) && txBlockTime.After(projection.phaseTwoStartTime) {
		if err := crossfireValidatorsView.UpdateTaskForOperatorAddress(
			constants.TASK_PHASE_2_PROPOSAL_VOTE_COLUMN_NAME,
			constants.COMPLETED,
			voterAddress,
		); err != nil {
			return fmt.Errorf("error updating validator Phase_2 Task_1 as completed s%v", err)
		}
	}

	if txBlockTime.After(projection.phaseThreeStartTime) {
		if err := crossfireValidatorsView.UpdateTaskForOperatorAddress(
			constants.TASK_PHASE_2_PROPOSAL_VOTE_COLUMN_NAME,
			constants.MISSED,
			voterAddress,
		); err != nil {
			return fmt.Errorf("error updating validator Phase_2 Task_1 as missed: s%v", err)
		}
	}

	return nil
}

// Update Tx sent count for sender
func (projection *Crossfire) updateTxSentCount(
	crossfireValidatorStatsView *view.CrossfireValidatorsStats,
	blockTime utctime.UTCTime,
	msgTransactionCreated *event_usecase.TransactionCreated,
) error {
	var phaseNumberPrefix string
	if blockTime.After(projection.phaseOneStartTime) && blockTime.Before(projection.phaseTwoStartTime) {
		phaseNumberPrefix = constants.PHASE_1_TX_SENT_PREFIX
	} else if blockTime.After(projection.phaseTwoStartTime) && blockTime.Before(projection.phaseThreeStartTime) {
		phaseNumberPrefix = constants.PHASE_2_TX_SENT_PREFIX
	} else if blockTime.After(projection.phaseThreeStartTime) && blockTime.Before(projection.competitionEndTime) {
		phaseNumberPrefix = constants.PHASE_3_TX_SENT_PREFIX
	}
	for _, sender := range msgTransactionCreated.Senders {

		// Only considering Pubkey address for now
		if sender.Type == constants.TYPE_URL_PUBKEY && sender.MaybeThreshold == nil {

			// Increment count for address as per PHASE
			phaseAddressCountDbKey := phaseNumberPrefix + constants.DB_KEY_SEPARATOR + sender.Pubkeys[0]
			errIncrementing := crossfireValidatorStatsView.Increment(phaseAddressCountDbKey, 1)
			if errIncrementing != nil {
				return fmt.Errorf("error Phase wise tx sent count increment: s%v", errIncrementing)
			}

			// Increment TOTAL count for address
			totalAddressCountDbKey := constants.TOTAL_TX_SENT_PREFIX + constants.DB_KEY_SEPARATOR + sender.Pubkeys[0]
			errIncrementingTotal := crossfireValidatorStatsView.Increment(totalAddressCountDbKey, 1)
			if errIncrementingTotal != nil {
				return fmt.Errorf("error Incrementing tx sent count: s%v", errIncrementingTotal)
			}

		}
	}
	return nil
}