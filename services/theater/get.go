package theater

import (
	"context"
	"github.com/CastyLab/grpc.proto"
	"github.com/CastyLab/grpc.proto/messages"
	"github.com/CastyLab/grpc.server/db"
	"github.com/CastyLab/grpc.server/db/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"net/http"
	"time"
)

func (s *Service) GetTheater(ctx context.Context, theater *messages.Theater) (*proto.UserTheaterResponse, error) {

	var (
		collection     = db.Connection.Collection("theaters")
		failedResponse = &proto.UserTheaterResponse{
			Status:  "failed",
			Code:    http.StatusInternalServerError,
			Message: "Could not get theater, Please try again later!",
		}
	)

	if theater.Id == "" {
		return &proto.UserTheaterResponse{
			Status:  "failed",
			Code:    420,
			Message: "Validation error, TheaterId is required!",
		}, nil
	}

	objectId, _ := primitive.ObjectIDFromHex(theater.Id)

	filter := bson.M{
		"$or": []interface{} {
			bson.M{"hash": theater.Hash},
			bson.M{"_id": objectId},
		},
	}

	mCtx, _ := context.WithTimeout(ctx, 20 * time.Second)

	var dbTheater = new(models.Theater)
	if err := collection.FindOne(mCtx, filter).Decode(dbTheater); err != nil {
		return failedResponse, nil
	}

	theater, err := SetDbTheaterToMessageTheater(mCtx, dbTheater)
	if err != nil {
		return failedResponse, nil
	}

	return &proto.UserTheaterResponse{
		Status:  "success",
		Code:    http.StatusOK,
		Result:  theater,
	}, nil
}
