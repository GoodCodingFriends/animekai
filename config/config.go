package config

type Config struct {
	Env                Env    `envconfig:"ENV" default:"dev"`
	AnnictToken        string `envconfig:"ANNICT_TOKEN" required:"true"`
	AnnictEndpoint     string `envconfig:"ANNICT_ENDPOINT" required:"true"`
	SlackSigningSecret string `envconfig:"SLACK_SIGNING_SECRET" required:"true"`
}

type Env string

func (e Env) IsDev() bool {
	return string(e) == "dev"
}

func (e Env) IsProd() bool {
	return string(e) == "prod"
}
