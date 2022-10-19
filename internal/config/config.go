package config

import (
	"fmt"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/pkg/errors"
	"github.com/spf13/viper"
)

type Config struct {
	PGConn   *pgxpool.Config `yaml:"pg_conn"`
	TGConfig []tgConfig      `yaml:"token"`
	Server   *Server         `yaml:"server"`
}

type tgConfig struct {
	BotLang string `yaml:"bot_lang"`
	BotLink string `yaml:"link"`
	Token   string `yaml:"token"`
}

type Server struct {
	Ip        string `yaml:"ip"`
	GetAdmins string `yaml:"get_admins"`
	GetInfo   string `yaml:"get_info"`
}

func InitConfig() (*Config, string, error) {
	vp := viper.New()

	vp.AddConfigPath("config")
	vp.SetConfigName("config")

	if err := vp.ReadInConfig(); err != nil {
		return nil, "", err
	}

	var config Config

	connString := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?pool_max_conns=%d",
		vp.Get("db_conn_config.user"),
		vp.Get("db_conn_config.password"),
		vp.Get("db_conn_config.host"),
		vp.Get("db_conn_config.port"),
		vp.Get("db_conn_config.db_name"),
		vp.Get("db_conn_config.pool_max_conns"))

	connForMigrations := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		vp.Get("db_conn_config.host"),
		vp.Get("db_conn_config.port"),
		vp.Get("db_conn_config.user"),
		vp.Get("db_conn_config.password"),
		vp.Get("db_conn_config.db_name"))

	fmt.Println(connString)
	pgxConfig, err := pgxpool.ParseConfig(connString)
	if err != nil {
		return nil, "", errors.Wrap(err, "`Init config` failed to parse config")
	}

	config.TGConfig = []tgConfig{{
		BotLang: vp.GetString("tg_config.0.0.bot_lang"),
		BotLink: vp.GetString("tg_config.0.0.link"),
		Token:   vp.GetString("tg_config.0.0.token"),
	},
		{
			BotLang: vp.GetString("tg_config.1.1.bot_lang"),
			BotLink: vp.GetString("tg_config.1.1.link"),
			Token:   vp.GetString("tg_config.1.1.token"),
		},
	}

	config.Server = &Server{
		Ip:        vp.GetString("server.ip"),
		GetAdmins: vp.GetString("server.routes.0"),
		GetInfo:   vp.GetString("server.routes.1"),
	}

	config.PGConn = pgxConfig

	return &config, connForMigrations, nil
}
