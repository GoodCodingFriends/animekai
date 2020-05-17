package config

type Config struct {
	Env            string `envconfig:"ENV" default:"dev"`
	AnnictToken    string `envconfig:"ANNICT_TOKEN" required:"true"`
	AnnictEndpoint string `envconfig:"ANNICT_ENDPOINT" required:"true"`
}
