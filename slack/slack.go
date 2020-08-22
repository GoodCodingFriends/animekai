package slack

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"path"
	"strconv"
	"strings"

	"github.com/GoodCodingFriends/animekai/annict"
	"github.com/GoodCodingFriends/animekai/resource"
	"github.com/grpc-ecosystem/go-grpc-middleware/logging/zap/ctxzap"
	"github.com/morikuni/failure"
	"github.com/slack-go/slack"
	"go.uber.org/zap"
)

type commandHandler struct {
	logger        *zap.Logger
	signingSecret string
	webhookURL    string

	annict annict.Service
}

func NewCommandHandler(logger *zap.Logger, signingSecret, webhookURL string, annictService annict.Service) http.Handler {
	return &commandHandler{
		logger:        logger,
		signingSecret: signingSecret,
		webhookURL:    webhookURL,
		annict:        annictService,
	}
}

// TODO: Use failure.Code.
func (h *commandHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.logger.Warn("non-POST request")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	verifier, err := slack.NewSecretsVerifier(r.Header, h.signingSecret)
	if err != nil {
		h.logger.Warn("failed to create a new secrets verifier", zap.Error(err))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	r.Body = ioutil.NopCloser(io.TeeReader(r.Body, &verifier))

	cmd, err := slack.SlashCommandParse(r)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		h.logger.Warn("failed to parse slash command", zap.Error(err))
		return
	}

	if err := verifier.Ensure(); err != nil { //nolint:govet
		w.WriteHeader(http.StatusBadRequest)
		h.logger.Warn("failed to authenticate request", zap.Error(err))
		return
	}

	sp := strings.Split(cmd.Text, " ")
	h.handle(sp) // handle runs asynchronously.

	w.Header().Set("Content-Type", "application/json")
	params := &slack.Msg{ResponseType: slack.ResponseTypeInChannel}
	if err := json.NewEncoder(w).Encode(params); err != nil {
		h.logger.Warn("failed to encode response body", zap.Error(err))
	}
}

func (h *commandHandler) handle(args []string) {
	go func() {
		msg := func() string {
			switch args[0] {
			case "start":
				h.logger.Info("start")
				episodes, err := start(ctxzap.ToContext(context.Background(), h.logger.Named("start")), h.annict)
				if err != nil {
					h.logger.Error("failed to call start", zap.Error(err))
					return ""
				}

				var text string
				for _, e := range episodes {
					text += fmt.Sprintf("- %s %s %s\n", e.WorkTitle, e.NumberText, e.Title)
				}

				return text
			case "add":
				h.logger.Info("add")
				if len(args) == 1 || args[1] == "-h" || args[1] == "--help" {
					return "usage: /animekai add https://annict.jp/works/<workID>"
				}
				ctx := ctxzap.ToContext(context.Background(), h.logger.Named("add"))
				if err := add(ctx, h.annict, args[1:]); err != nil {
					return ""
				}
				return ":lgtm-1:"
			}
			return ""
		}()

		err := slack.PostWebhook(
			h.webhookURL,
			&slack.WebhookMessage{
				Text: msg,
			},
		)
		if err != nil {
			h.logger.Error("failed to process command", zap.Error(err))
		}
	}()
}

func start(ctx context.Context, annictService annict.Service) ([]*resource.Episode, error) {
	episodes, err := annictService.CreateNextEpisodeRecords(ctx)
	if err != nil {
		return nil, failure.Wrap(err)
	}
	return episodes, nil
}

func add(ctx context.Context, annictService annict.Service, args []string) error {
	v := path.Base(args[0])
	workID, err := strconv.Atoi(v)
	if err != nil {
		return failure.Wrap(err, failure.Context{"work_id": args[0]})
	}

	if err := annictService.UpdateWorkStatus(ctx, workID, annict.StatusStateWatching); err != nil {
		return failure.Wrap(err)
	}
	return nil
}
