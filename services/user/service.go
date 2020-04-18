package user

import (
	"context"
	"github.com/CastyLab/grpc.proto/proto"
	"github.com/CastyLab/grpc.server/db"
	"github.com/CastyLab/grpc.server/helpers"
	"github.com/CastyLab/grpc.server/services/auth"
	"github.com/getsentry/sentry-go"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"net/http"
)

type Service struct {}

func (s *Service) RemoveActivity(ctx context.Context, req *proto.AuthenticateRequest) (*proto.Response, error) {

	user, err := auth.Authenticate(req)
	if err != nil {
		return &proto.Response{
			Status:  "failed",
			Code:    http.StatusUnauthorized,
			Message: "Unauthorized!",
		}, nil
	}

	var (
		filter = bson.M{"_id": user.ID}
		update = bson.M{
			"$set": bson.M{
				"activity": bson.M{},
			},
		}
	)

	if _, err := db.Connection.Collection("users").UpdateOne(ctx, filter, update); err != nil {
		sentry.CaptureException(err)
		return &proto.Response{
			Status:  "failed",
			Code:    http.StatusInternalServerError,
			Message: "The requested parameter is not updated!",
		}, nil
	}

	return &proto.Response{
		Status:  "success",
		Code:    http.StatusOK,
		Message: "The requested parameter is updated successfully!",
	}, nil
}

func (s *Service) UpdateActivity(ctx context.Context, req *proto.UpdateActivityRequest) (*proto.Response, error) {

	user, err := auth.Authenticate(req.AuthRequest)
	if err != nil {
		return &proto.Response{
			Status:  "failed",
			Code:    http.StatusUnauthorized,
			Message: "Unauthorized!",
		}, nil
	}

	activityObjectId, err := primitive.ObjectIDFromHex(req.Activity.Id)
	if err != nil {

		return &proto.Response{
			Status:  "failed",
			Code:    http.StatusNotAcceptable,
			Message: "Activity id is invalid!",
		}, nil
	}

	var (
		filter = bson.M{"_id": user.ID}
		update = bson.M{
			"$set": bson.M{
				"activity": bson.M{
					"_id": activityObjectId,
					"activity": req.Activity.Activity,
				},
			},
		}
	)

	if _, err := db.Connection.Collection("users").UpdateOne(ctx, filter, update); err != nil {
		sentry.CaptureException(err)
		return &proto.Response{
			Status:  "failed",
			Code:    http.StatusInternalServerError,
			Message: "The requested parameter is not updated!",
		}, nil
	}

	return &proto.Response{
		Status:  "success",
		Code:    http.StatusOK,
		Message: "The requested parameter is updated successfully!",
	}, nil
}

func (s *Service) UpdateState(ctx context.Context, req *proto.UpdateStateRequest) (*proto.Response, error) {

	user, err := auth.Authenticate(req.AuthRequest)
	if err != nil {
		return &proto.Response{
			Status:  "failed",
			Code:    http.StatusUnauthorized,
			Message: "Unauthorized!",
		}, nil
	}

	var (
		filter = bson.M{"_id": user.ID}
		update = bson.M{
			"$set": bson.M{
				"state": int(req.State),
			},
		}
	)

	if _, err := db.Connection.Collection("users").UpdateOne(ctx, filter, update); err != nil {
		sentry.CaptureException(err)
		return &proto.Response{
			Status:  "failed",
			Code:    http.StatusInternalServerError,
			Message: "The requested parameter is not updated!",
		}, nil
	}

	return &proto.Response{
		Status:  "success",
		Code:    http.StatusOK,
		Message: "The requested parameter is updated successfully!",
	}, nil
}

func (s *Service) GetUser(ctx context.Context, req *proto.AuthenticateRequest) (*proto.GetUserResponse, error) {

	user, err := auth.Authenticate(req)
	if err != nil {
		return nil, err
	}

	protoUser, err := helpers.SetDBUserToProtoUser(user)
	if err != nil {
		sentry.CaptureException(err)
		return &proto.GetUserResponse{
			Message: "Could not decode user!",
			Status: "failed",
			Code:   http.StatusInternalServerError,
		}, nil
	}

	return &proto.GetUserResponse{
		Result: protoUser,
		Status: "success",
		Code:   http.StatusOK,
	}, nil
}