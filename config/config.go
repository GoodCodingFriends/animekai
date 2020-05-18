package config

type Config struct {
	Env            Env    `envconfig:"ENV" default:"dev"`
	AnnictToken    string `envconfig:"ANNICT_TOKEN" required:"true"`
	AnnictEndpoint string `envconfig:"ANNICT_ENDPOINT" required:"true"`
}

type Env string

func (e Env) IsDev() bool {
	return string(e) == "dev"
}
