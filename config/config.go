package config

type Config struct {
	AnnictToken    string `envconfig:"ANNICT_TOKEN" required:"true"`
	AnnictEndpoint string `envconfig:"ANNICT_ENDPOINT" required:"true"`
}
