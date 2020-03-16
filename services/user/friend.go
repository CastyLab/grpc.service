package user

import (
	"context"
	"github.com/CastyLab/grpc.proto"
	"github.com/CastyLab/grpc.proto/messages"
	"github.com/CastyLab/grpc.server/db"
	"github.com/CastyLab/grpc.server/db/models"
	"github.com/CastyLab/grpc.server/services/auth"
	"github.com/golang/protobuf/ptypes"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"net/http"
	"time"
)

func (s *Service) GetFriend(ctx context.Context, req *proto.FriendRequest) (*proto.FriendResponse, error) {

	var (

		database   = db.Connection

		dbFriend           = new(models.Friend)
		dbFriendUserObject = new(models.User)

		mCtx, _  = context.WithTimeout(ctx, 20 * time.Second)

		userCollection    = database.Collection("users")
		friendsCollection = database.Collection("friends")

		failedResponse = &proto.FriendResponse{
			Status:  "failed",
			Code:    http.StatusInternalServerError,
			Message: "Could not get the friend, Please try again later!",
		}
	)

	user, err := auth.Authenticate(req.AuthRequest)
	if err != nil {
		return &proto.FriendResponse{
			Status:  "failed",
			Code:    http.StatusUnauthorized,
			Message: "Unauthorized!",
		}, nil
	}

	if err := userCollection.FindOne(mCtx, bson.M{ "username": req.FriendId }).Decode(dbFriendUserObject); err != nil {
		return failedResponse, nil
	}

	filter := bson.M{
		"accepted": true,
		"$or": []interface{}{
			bson.M{
				"friend_id": user.ID,
				"user_id": dbFriendUserObject.ID,
			},
			bson.M{
				"user_id": user.ID,
				"friend_id": dbFriendUserObject.ID,
			},
		},
	}

	if err := friendsCollection.FindOne(mCtx, filter).Decode(dbFriend); err != nil {
		return failedResponse, nil
	}

	friendUser, err := SetDBUserToProtoUser(dbFriendUserObject)
	if err != nil {
		return failedResponse, nil
	}

	return &proto.FriendResponse{
		Status:  "success",
		Code:    http.StatusOK,
		Result:  friendUser,
	}, nil
}

func (s *Service) GetFriendRequest(ctx context.Context, req *proto.FriendRequest) (*messages.Friend, error) {

	var (
		database   = db.Connection
		dbFriend   = new(models.Friend)
		mCtx, _    = context.WithTimeout(ctx, 20 * time.Second)
		friendsCollection = database.Collection("friends")
		failedResponse = &messages.Friend{}
	)

	if _, err := auth.Authenticate(req.AuthRequest); err != nil {
		return failedResponse, err
	}

	requestObjectId, err := primitive.ObjectIDFromHex(req.RequestId)
	if err != nil {
		return failedResponse, err
	}

	if err := friendsCollection.FindOne(mCtx, bson.M{ "_id": requestObjectId }).Decode(dbFriend); err != nil {
		return failedResponse, err
	}

	friendUser, err := SetDBFRToProto(dbFriend)
	if err != nil {
		return failedResponse, err
	}

	return friendUser, nil
}

func SetDBFRToProto(friend *models.Friend) (*messages.Friend, error) {

	createdAt,  _ := ptypes.TimestampProto(friend.CreatedAt)
	updatedAt, _ := ptypes.TimestampProto(friend.UpdatedAt)

	protoUser := &messages.Friend{
		Accepted:  friend.Accepted,
		CreatedAt: createdAt,
		UpdatedAt: updatedAt,
	}

	return protoUser, nil

}
