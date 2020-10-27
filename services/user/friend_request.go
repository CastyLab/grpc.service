package user

import (
	"context"
	"github.com/CastyLab/grpc.proto/proto"
	"github.com/CastyLab/grpc.proto/protocol"
	"github.com/CastyLab/grpc.server/db"
	"github.com/CastyLab/grpc.server/db/models"
	"github.com/CastyLab/grpc.server/helpers"
	"github.com/CastyLab/grpc.server/services/auth"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"log"
	"net/http"
	"time"
)

func (s *Service) GetPendingFriendRequests(ctx context.Context, req *proto.AuthenticateRequest) (*proto.PendingFriendRequests, error) {

	var (
		friendRequests    = make([]*proto.FriendRequest, 0)
		userCollection    = db.Connection.Collection("users")
		friendsCollection = db.Connection.Collection("friends")
	)

	user, err := auth.Authenticate(req)
	if err != nil {
		return nil, err
	}

	filter := bson.M{
		"accepted": false,
		"friend_id": user.ID,
	}

	cursor, err := friendsCollection.Find(ctx, filter)
	if err != nil {
		return nil, status.Error(codes.NotFound, "Could not find pending friends!")
	}

	for cursor.Next(ctx) {

		var friend = new(models.Friend)
		if err := cursor.Decode(friend); err != nil {
			continue
		}

		filter := bson.M{"_id": friend.UserId}
		dbFriend := new(models.User)

		if err := userCollection.FindOne(ctx, filter).Decode(dbFriend); err != nil {
			continue
		}

		friendRequests = append(friendRequests, &proto.FriendRequest{
			RequestId: friend.ID.Hex(),
			Friend:    helpers.NewProtoUser(dbFriend),
		})
	}

	return &proto.PendingFriendRequests{
		Status:  "success",
		Code:    http.StatusOK,
		Result:  friendRequests,
	}, nil
}

func (s *Service) AcceptFriendRequest(ctx context.Context, req *proto.FriendRequest) (*proto.Response, error) {

	var (
		friendRequest     = new(models.Friend)
		friendsCollection = db.Connection.Collection("friends")
		notifsCollection  = db.Connection.Collection("notifications")
		failedResponse    = status.Error(codes.Internal, "Could not accept friend request, Please try again later!")
	)

	user, err := auth.Authenticate(req.AuthRequest)
	if err != nil {
		return nil, err
	}
	protoUser := helpers.NewProtoUser(user)

	frObjectID, err := primitive.ObjectIDFromHex(req.RequestId)
	if err != nil {
		return nil, failedResponse
	}

	filter := bson.M{
		"_id": frObjectID,
		"$or": []interface{}{
			bson.M{"friend_id": user.ID},
			bson.M{"user_id": user.ID},
		},
	}

	if err := friendsCollection.FindOne(ctx, filter).Decode(&friendRequest); err != nil {
		return nil, status.Error(codes.NotFound, "Could not find friend request!")
	}

	if friendRequest.Accepted {
		return nil, status.Error(codes.InvalidArgument, "Friend request is not valid anymore!")
	}

	findNotif := bson.M{
		"extra": friendRequest.ID,
		"to_user_id": user.ID,
	}

	// update user's notification to read
	_, _ = notifsCollection.UpdateOne(ctx, findNotif, bson.M{
		"$set": bson.M{
			"read": true,
			"updated_at": time.Now(),
			"read_at": time.Now(),
		},
	})

	var (
		update = bson.M{
			"$set": bson.M{
				"accepted": true,
			},
		}
		updateFilter = bson.M{
			"_id": friendRequest.ID,
		}
	)

	updateResult, err := friendsCollection.UpdateOne(ctx, updateFilter, update)
	if err != nil {
		return nil, failedResponse
	}

	if updateResult.ModifiedCount == 1 {

		// send event to self user clients
		pms := &proto.FriendRequestAcceptedMsgEvent{Friend: protoUser}
		buffer, err := protocol.NewMsgProtobuf(proto.EMSG_SELF_FRIEND_REQUEST_ACCEPTED, pms)
		if err == nil {
			if err := helpers.SendEventToUser(ctx, buffer.Bytes(), protoUser); err != nil {
				log.Println(err)
			}
		}

		// send event to friend clients
		friendID := friendRequest.FriendId.Hex()
		if friendRequest.FriendId.Hex() == user.ID.Hex() {
			friendID = friendRequest.UserId.Hex()
		}
		if buffer, err := protocol.NewMsgProtobuf(proto.EMSG_FRIEND_REQUEST_ACCEPTED, pms); err == nil {
			if err := helpers.SendEventToUser(ctx, buffer.Bytes(), &proto.User{Id: friendID}); err != nil {
				log.Println(err)
			}
		}

		return &proto.Response{
			Status:  "success",
			Code:    http.StatusOK,
			Message: "Friend request accepted successfully!",
		}, nil
	}

	return nil, failedResponse
}

func (s *Service) SendFriendRequest(ctx context.Context, req *proto.FriendRequest) (*proto.Response, error) {

	var (
		database                = db.Connection
		friend                  = new(models.User)
		userCollection          = database.Collection("users")
		friendsCollection       = database.Collection("friends")
		notificationsCollection = database.Collection("notifications")
		failedResponse          = status.Error(codes.Internal, "Could not create friend request, Please try again later!")
	)

	user, err := auth.Authenticate(req.AuthRequest)
	if err != nil {
		return nil, err
	}

	friendObjectId, err := primitive.ObjectIDFromHex(req.FriendId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "Invalid friend id!")
	}

	var (
		filterFr = bson.M{
			"$or": []interface{}{
				bson.M{
					"friend_id": user.ID,
					"user_id": friendObjectId,
				},
				bson.M{
					"user_id": user.ID,
					"friend_id": friendObjectId,
				},
			},
		}
	)

	alreadyFriendsCount, err := friendsCollection.CountDocuments(ctx, filterFr)
	if err != nil {
		return nil, failedResponse
	}

	if alreadyFriendsCount != 0 {
		return nil, status.Error(codes.Aborted, "Friend request sent already!")
	}

	if err := userCollection.FindOne(ctx, bson.M{"_id": friendObjectId}).Decode(friend); err != nil {
		return nil, failedResponse
	}

	friendRequest := bson.M{
		"friend_id": friend.ID,
		"user_id":   user.ID,
		"accepted":  false,
	}

	friendrequestInsertData, err := friendsCollection.InsertOne(ctx, friendRequest)
	if err != nil {
		return nil, failedResponse
	}

	frInsertID := friendrequestInsertData.InsertedID.(primitive.ObjectID)

	notification := bson.M{
		"type":         int64(proto.Notification_NEW_FRIEND),
		"read":         false,
		"from_user_id": user.ID,
		"to_user_id":   friend.ID,
		"extra":        frInsertID,
		"read_at":      time.Now(),
		"created_at":   time.Now(),
		"updated_at":   time.Now(),
	}

	if _, err := notificationsCollection.InsertOne(ctx, notification); err != nil {
		return nil, failedResponse
	}

	// send event to friend clients
	buffer, err := protocol.NewMsgProtobuf(proto.EMSG_NEW_NOTIFICATION, &proto.NotificationMsgEvent{})
	if err == nil {
		if err := helpers.SendEventToUser(ctx, buffer.Bytes(), &proto.User{Id: friend.ID.Hex()}); err != nil {
			log.Println(err)
		}
	}

	return &proto.Response{
		Status:  "success",
		Code:    http.StatusOK,
		Message: "Friend request added successfully!",
	}, nil
}
