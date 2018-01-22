package grpc

import (
	"context"

	"github.com/pkg/errors"
	"github.com/stairlin/lego/ctx/app"
	"github.com/stairlin/lego/ctx/journey"
	"google.golang.org/grpc/metadata"
)

const contextMD = "lego-journey-bin"

// ErrMissingJourney occurs when there is an attempt to extract a journey
// from an incomming context that does not have it
var ErrMissingJourney = errors.New("missing journey")

// ExtractContext extracts journey from a generic context
func ExtractContext(context context.Context, app app.Ctx) (journey.Ctx, error) {
	md, ok := metadata.FromIncomingContext(context)
	if !ok {
		return nil, errors.New("missing metadata")
	}
	data, ok := md[contextMD]
	if !ok {
		return nil, ErrMissingJourney
	}
	ctx, err := journey.UnmarshalGob(app, []byte(data[0]))
	if err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal Context")
	}
	ctx = ctx.BranchOff(journey.Child)
	return ctx, nil
}

// EmbedContext embeds a journey into a generic context
func EmbedContext(ctx context.Context) (context.Context, error) {
	j, ok := ctx.(journey.Ctx)
	if !ok {
		return ctx, nil
	}
	data, err := journey.MarshalGob(j)
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal Context")
	}
	md := metadata.MD{}
	md[contextMD] = append(md[contextMD], string(data))
	return metadata.NewOutgoingContext(ctx, md), nil
}
