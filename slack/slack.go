package slack

import (
	"context"
	"io"
	"io/ioutil"
	"net/http"

	"github.com/GoodCodingFriends/animekai/annict"
	"github.com/morikuni/failure"
	"github.com/slack-go/slack"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
)

type commandHandler struct {
	logger        *zap.Logger
	signingSecret string

	annict annict.Service
}

func NewCommandHandler(logger *zap.Logger, signingSecret string) http.Handler {
	return &commandHandler{
		logger:        logger,
		signingSecret: signingSecret,
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

	if err := verifier.Ensure(); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		h.logger.Warn("failed to authenticate request", zap.Error(err))
		return
	}

	switch cmd.Text {
	case "start":
		h.logger.Info("start")
	case "add":
		h.logger.Info("add")
	}
}

func start(ctx context.Context, annictService annict.Service) error {
	works, _, err := annictService.ListWorks(ctx, annict.WorkStateWatching, "", 100)
	if err != nil {
		return failure.Wrap(err)
	}

	var eg errgroup.Group
	for _, work := range works {
		eg.Go(func() error {
			return annictService.CreateNextEpisodeRecord(ctx, work.Id)
		})
	}

	if err := eg.Wait(); err != nil {
		return failure.Wrap(err)
	}
}
