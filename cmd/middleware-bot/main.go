package main

import (
	"github.com/BlackRRR/middleware-bot/internal/app/model"
	"github.com/BlackRRR/middleware-bot/internal/app/repository"
	"github.com/BlackRRR/middleware-bot/internal/app/services"
	"github.com/BlackRRR/middleware-bot/internal/app/services/bot"
	"github.com/BlackRRR/middleware-bot/internal/app/utils"
	"github.com/BlackRRR/middleware-bot/internal/config"
	"github.com/BlackRRR/middleware-bot/internal/db"
	"github.com/BlackRRR/middleware-bot/internal/log"
	"github.com/bots-empire/base-bot/msgs"
	log2 "log"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	//init logger
	logger := log.NewDefaultLogger().Prefix("Middleware Bot")
	log.PrintLogo("Middleware Bot", []string{"8000FF"})

	//Init Config
	cfg, err := config.InitConfig()
	if err != nil {
		log2.Fatal(err)
	}

	//Init Database
	pool, err := db.InitDB(cfg.PGConn)
	if err != nil {
		log2.Fatal(err)
	}

	//init bots config
	srvs := make([]*services.Services, 0)
	for i := range cfg.Token {
		globalBot := model.FillBotsConfig(cfg.Token[i], cfg.BotLink, cfg.BotLang)

		//init msgs service
		msgsSrv := msgs.NewService(globalBot, []int64{})

		//Init Repository
		repo := repository.NewRepository(pool, msgsSrv, globalBot)

		//Init Services
		initServices := services.InitServices(repo, msgsSrv, globalBot)

		srvs = append(srvs, initServices)
	}

	for _, service := range srvs {
		go func(handler *bot.BotService) {
			handler.ActionsWithUpdates(logger, utils.NewSpreader(time.Minute))
		}(service.BotSrv)

		service.BotSrv.BaseBotSrv.SendNotificationToDeveloper("Bot are restart", false)

		logger.Ok("All handlers are running")
	}

	sig := <-subscribeToSystemSignals()

	log2.Printf("shutdown all process on '%s' system signal\n", sig.String())
}

func subscribeToSystemSignals() chan os.Signal {
	ch := make(chan os.Signal, 10)
	signal.Notify(ch,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT,
		syscall.SIGHUP,
	)
	return ch
}
