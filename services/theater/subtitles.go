package theater

import (
	"context"
	"github.com/CastyLab/grpc.proto/proto"
	"github.com/CastyLab/grpc.server/db"
	"github.com/CastyLab/grpc.server/db/models"
	"github.com/CastyLab/grpc.server/helpers"
	"github.com/CastyLab/grpc.server/services/auth"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"net/http"
)

// Get all subtitles from theater
func (s *Service) GetSubtitles(ctx context.Context, req *proto.TheaterAuthRequest) (*proto.TheaterSubtitlesResponse, error) {

	var (
		theater        = new(models.Theater)
		subtitles      = make([]*proto.Subtitle, 0)
		collection     = db.Connection.Collection("subtitles")
		failedResponse = status.Error(codes.Internal, "Could not get subtitles, Please try again later!")
	)

	if _, err := auth.Authenticate(req.AuthRequest); err != nil {
		return nil, err
	}

	var (
		theaterObjectID, _ = primitive.ObjectIDFromHex(req.Theater.Id)
		findFilter = bson.M{ "_id": theaterObjectID }
	)

	if err := db.Connection.Collection("theaters").FindOne(ctx, findFilter).Decode(theater); err != nil {
		return nil, status.Error(codes.NotFound, "Could not find theater!")
	}

	cursor, err := collection.Find(ctx, bson.M{"theater_id": theaterObjectID})
	if err != nil {
		return nil, failedResponse
	}

	for cursor.Next(ctx) {
		subtitle := new(models.Subtitle)
		if err := cursor.Decode(subtitle); err != nil {
			continue
		}
		protoMsg, err := helpers.NewSubtitleProto(subtitle)
		if err != nil {
			continue
		}
		subtitles = append(subtitles, protoMsg)
	}

	return &proto.TheaterSubtitlesResponse{
		Status:  "success",
		Code:    http.StatusOK,
		Result:  subtitles,
	}, nil
}

// Remove subtitle from theater
func (s *Service) RemoveSubtitle(ctx context.Context, req *proto.RemoveSubtitleRequest) (*proto.Response, error) {

	var (
		theater        = new(models.Theater)
		collection     = db.Connection.Collection("subtitles")
		failedResponse = status.Error(codes.Internal, "Could not remove subtitle, Please try again later!")
	)

	user, err := auth.Authenticate(req.AuthRequest)
	if err != nil {
		return nil, err
	}

	var (
		theaterObjectID, _ = primitive.ObjectIDFromHex(req.Subtitle.TheaterId)
		findFilter = bson.M{
			"_id": theaterObjectID,
			"user_id": user.ID,
		}
	)

	if err := db.Connection.Collection("theaters").FindOne(ctx, findFilter).Decode(theater); err != nil {
		return nil, status.Error(codes.NotFound, "Could not find theater!")
	}

	var (
		subtitleObjectID, _ = primitive.ObjectIDFromHex(req.Subtitle.Id)
		filter = bson.M{
			"_id": subtitleObjectID,
			"theater_id": theaterObjectID,
		}
	)

	result, err := collection.DeleteOne(ctx, filter)
	if err != nil || result.DeletedCount != 1 {
		return nil, failedResponse
	}

	return &proto.Response{
		Status:  "success",
		Code:    http.StatusOK,
		Message: "Subtitle deleted successfully!",
	}, nil
}
