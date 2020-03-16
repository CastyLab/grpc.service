package user

import (
	"context"
	proto "github.com/CastyLab/grpc.proto"
	"github.com/CastyLab/grpc.proto/messages"
	"github.com/CastyLab/grpc.server/db"
	"github.com/CastyLab/grpc.server/db/models"
	"github.com/CastyLab/grpc.server/services/auth"
	"github.com/golang/protobuf/ptypes"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
	"net/http"
	"time"
)

func SetDBNotificationToProto(notif *models.Notification) (*messages.Notification, error) {

	var (
		readAt,  _     = ptypes.TimestampProto(notif.ReadAt)
		createdAt,  _  = ptypes.TimestampProto(notif.CreatedAt)
		updatedAt, _   = ptypes.TimestampProto(notif.UpdatedAt)

		fromUser = new(models.User)
		mCtx, _  = context.WithTimeout(context.Background(), 10 * time.Second)
	)

	cursor := db.Connection.Collection("users").FindOne(mCtx, bson.M{ "_id": notif.FromUserId })
	if err := cursor.Decode(&fromUser); err != nil {
		return nil, err
	}

	protoUser, err := SetDBUserToProtoUser(fromUser)
	if err != nil {
		return nil, err
	}

	return &messages.Notification{
		Id:         notif.ID.Hex(),
		Type:       notif.Type,
		Extra:      notif.Extra.Hex(),
		Read:       notif.Read,
		ReadAt:     readAt,
		FromUser:   protoUser,
		CreatedAt:  createdAt,
		UpdatedAt:  updatedAt,
	}, nil
}

type NotificationData struct {
	Data   string   `json:"data"`
	User   string   `json:"user"`
}

func (s *Service) CreateNotification(ctx context.Context, req *proto.CreateNotificationRequest) (*proto.NotificationResponse, error) {

	var (
		mCtx, _ = context.WithTimeout(ctx, 20 * time.Second)
		collection     = db.Connection.Collection("notifications")
		failedResponse = &proto.NotificationResponse{
			Status:  "failed",
			Code:    http.StatusInternalServerError,
			Message: "Could not create notification, Please try again later!",
		}
	)

	user, err := auth.Authenticate(req.AuthRequest)
	if err != nil {
		return &proto.NotificationResponse{
			Status:  "failed",
			Code:    http.StatusUnauthorized,
			Message: "Unauthorized!",
		}, nil
	}

	if req.Notification == nil {
		return &proto.NotificationResponse{
			Status:  "failed",
			Code:    420,
			Message: "Validation error, Notification entry not exists!",
		}, nil
	}

	friendObjectId, err := primitive.ObjectIDFromHex(req.Notification.ToUserId)
	if err != nil {
		return failedResponse, nil
	}

	notification := bson.M{
		"type":         int64(req.Notification.Type),
		"read":         req.Notification.Read,
		"from_user_id": user.ID,
		"to_user_id":   friendObjectId,
		"read_at":      time.Now(),
		"created_at":   time.Now(),
		"updated_at":   time.Now(),
	}

	switch req.Notification.Type {
	case messages.NOTIFICATION_TYPE_NEW_THEATER_INVITE:
		theaterObjectId, err := primitive.ObjectIDFromHex(req.Notification.Extra)
		if err != nil {
			return failedResponse, nil
		}
		notification["extra"] = theaterObjectId
	}

	if _, err := collection.InsertOne(mCtx, notification); err != nil {
		return failedResponse, nil
	}

	return &proto.NotificationResponse{
		Status:  "success",
		Code:    http.StatusOK,
		Message: "Notification created successfully!",
	}, nil
}

func (s *Service) GetNotifications(ctx context.Context, req *proto.AuthenticateRequest) (*proto.NotificationResponse, error) {

	var (
		notifications []*messages.Notification

		database   = db.Connection
		mCtx, _    = context.WithTimeout(ctx, 20 * time.Second)

		notificationCollection = database.Collection("notifications")

		failedResponse = &proto.NotificationResponse{
			Status:  "failed",
			Code:    http.StatusInternalServerError,
			UnreadCount: 0,
			Message: "Could not get notifications, Please try again later!",
		}
	)

	user, err := auth.Authenticate(req)
	if err != nil {
		return &proto.NotificationResponse{
			Status:  "failed",
			Code:    http.StatusUnauthorized,
			Message: "Unauthorized!",
			UnreadCount: 0,
		}, nil
	}

	qOpts := options.Find()
	qOpts.SetSort(bson.D{
		{"created_at", -1},
	})

	cursor, err := notificationCollection.Find(mCtx, bson.M{"to_user_id": user.ID}, qOpts)
	if err != nil {
		return failedResponse, nil
	}

	for cursor.Next(mCtx) {

		notification := new(models.Notification)
		if err := cursor.Decode(notification); err != nil {
			break
		}

		messageNotification, err := SetDBNotificationToProto(notification)
		if err != nil {
			break
		}

		notifications = append(notifications, messageNotification)
	}

	filter := bson.M{
		"to_user_id": user.ID,
		"read": false,
	}
	unreadCount, err := notificationCollection.CountDocuments(mCtx, filter)
	if err != nil {
		return failedResponse, nil
	}

	return &proto.NotificationResponse{
		Status:      "success",
		Code:        http.StatusOK,
		Result:      notifications,
		UnreadCount: unreadCount,
	}, nil
}

func (s *Service) ReadAllNotifications(ctx context.Context, req *proto.AuthenticateRequest) (*proto.NotificationResponse, error) {

	var (
		database   = db.Connection
		mCtx, _    = context.WithTimeout(ctx, 20 * time.Second)
		notificationCollection = database.Collection("notifications")
		failedResponse = &proto.NotificationResponse{
			Status:  "failed",
			Code:    http.StatusInternalServerError,
			UnreadCount: 0,
			Message: "Could not update notifications, Please try again later!",
		}
	)

	user, err := auth.Authenticate(req)
	if err != nil {
		return &proto.NotificationResponse{
			Status:  "failed",
			Code:    http.StatusUnauthorized,
			Message: "Unauthorized!",
			UnreadCount: 0,
		}, nil
	}

	var (
		filter = bson.M{"user_id": user.ID, "read": false}
		update = bson.M{"read": true}
	)

	if _, err := notificationCollection.UpdateMany(mCtx, filter, update); err != nil {
		return failedResponse, nil
	}

	return &proto.NotificationResponse{
		Status:   "success",
		Code:     http.StatusOK,
		Message:  "Notifications updated successfully!",
	}, nil
}