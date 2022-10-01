package config

import (
	"fmt"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/pkg/errors"
	"github.com/spf13/viper"
)

type Config struct {
	PGConn  *pgxpool.Config `yaml:"pg_conn"`
	BotLang string
	BotLink string
	Token   []string
}

func InitConfig() (*Config, error) {
	vp := viper.New()

	vp.AddConfigPath("config")
	vp.SetConfigName("config")

	if err := vp.ReadInConfig(); err != nil {
		return nil, err
	}

	var config Config

	connString := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?pool_max_conns=%s",
		vp.Get("config.db_conn_config.user"),
		vp.Get("config.db_conn_config.password"),
		vp.Get("config.db_conn_config.host"),
		vp.Get("config.db_conn_config.port"),
		vp.Get("config.db_conn_config.db_name"),
		vp.Get("config.db_conn_config.pool_max_conns"))

	pgxConfig, err := pgxpool.ParseConfig(connString)
	if err != nil {
		return nil, errors.Wrap(err, "`Init config` failed to parse config")
	}

	config.BotLang = vp.GetString("config.tg_config.bot_lang")
	config.BotLink = vp.GetString("config.tg_config.link")
	config.Token = vp.GetStringSlice("config.tg_config.token")

	config.PGConn = pgxConfig

	return &config, nil
}
