package config

type Config struct {
	Port               string `envconfig:"PORT" default:"8000"`
	Env                Env    `envconfig:"ENV" default:"dev"`
	AnnictToken        string `envconfig:"ANNICT_TOKEN" required:"true"`
	AnnictEndpoint     string `envconfig:"ANNICT_ENDPOINT" required:"true"`
	SlackSigningSecret string `envconfig:"SLACK_SIGNING_SECRET" required:"true"`
	SlackWebhookURL    string `envconfig:"SLACK_WEBHOOK_URL" required:"true"`
}

type Env string

func (e Env) IsDev() bool {
	return string(e) == "dev"
}

func (e Env) IsProd() bool {
	return string(e) == "prod"
}
