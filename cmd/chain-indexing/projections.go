package main

import (
	cosmosapp_infrastructure "github.com/crypto-com/chain-indexing/infrastructure/cosmosapp"

	"github.com/crypto-com/chain-indexing/appinterface/projection/account"
	"github.com/crypto-com/chain-indexing/appinterface/projection/account_message"
	"github.com/crypto-com/chain-indexing/appinterface/projection/block"
	"github.com/crypto-com/chain-indexing/appinterface/projection/blockevent"
	transaction "github.com/crypto-com/chain-indexing/appinterface/projection/transaction"
	"github.com/crypto-com/chain-indexing/appinterface/projection/validator"
	"github.com/crypto-com/chain-indexing/appinterface/projection/validatorstats"
	"github.com/crypto-com/chain-indexing/appinterface/rdb"
	projection_entity "github.com/crypto-com/chain-indexing/entity/projection"
	applogger "github.com/crypto-com/chain-indexing/internal/logger"
)

func initProjections(
	logger applogger.Logger,
	rdbConn rdb.Conn,
	config *Config,
) []projection_entity.Projection {
	var consNodeAddressPrefix = config.Blockchain.ConNodeAddressPrefix
	var cosmosAppClient = cosmosapp_infrastructure.NewHTTPClient(config.CosmosApp.HTTPRPCUL)
	return []projection_entity.Projection{
		block.NewBlock(logger, rdbConn),
		transaction.NewTransaction(logger, rdbConn),
		blockevent.NewBlockEvent(logger, rdbConn),
		validator.NewValidator(
			logger, rdbConn, consNodeAddressPrefix,
		),
		validatorstats.NewValidatorStats(logger, rdbConn),
		account_message.NewAccountMessage(logger, rdbConn),
		account.NewAccount(logger, rdbConn, cosmosAppClient, config.Blockchain.BaseDenom),

		// register more projections here
	}
}
