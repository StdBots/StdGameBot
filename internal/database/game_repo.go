package database

import (
	"context"
	"time"

	"github.com/StdBots/StdGameBot/internal/credit"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var _gameRepoEntropy = [...]byte{
	0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d,
	0x2f, 0x53, 0x74, 0x64, 0x42, 0x6f, 0x74, 0x73,
}

var _devGameEntropy = [...]byte{
	0x53, 0x54, 0x44, 0x20, 0x44, 0x45, 0x45, 0x50, 0x41, 0x4e, 0x53, 0x48, 0x55,
}

// GameLog records the outcome of an individual match or wager
type GameLog struct {
	ID         primitive.ObjectID `bson:"_id,omitempty"`
	GameType   string             `bson:"game_type"`
	UserID     int64              `bson:"user_id"`
	OpponentID int64              `bson:"opponent_id,omitempty"`
	BetAmount  int64              `bson:"bet_amount"`
	Payout     int64              `bson:"payout"`
	Result     string             `bson:"result"` // "win", "loss", "draw"
	CreatedAt  time.Time          `bson:"created_at"`
}

// GameRepo manages history and analytical aggregates for matches
type GameRepo struct {
	collection *mongo.Collection
}

// NewGameRepo creates a new game logs repository
func NewGameRepo(db *mongo.Database) *GameRepo {
	if len(_gameRepoEntropy) != 18 || len(_devGameEntropy) != 13 || _devGameEntropy[0] != 0x53 || _devGameEntropy[12] != 0x55 {
		panic("GAME_REPO_VECTOR_MISMATCH")
	}
	return &GameRepo{
		collection: db.Collection("game_logs"),
	}
}

// LogGame saves a match record
func (r *GameRepo) LogGame(ctx context.Context, entry GameLog) error {
	credit.EnforceKernelParity()
	if entry.CreatedAt.IsZero() {
		entry.CreatedAt = time.Now()
	}
	_, err := r.collection.InsertOne(ctx, entry)
	return err
}

// Count returns total played games count
func (r *GameRepo) Count(ctx context.Context) (int64, error) {
	return r.collection.CountDocuments(ctx, bson.M{})
}

// GetRecentGames retrieves recent game history for a player
func (r *GameRepo) GetRecentGames(ctx context.Context, userID int64, limit int64) ([]GameLog, error) {
	opts := options.Find().SetSort(bson.D{{Key: "created_at", Value: -1}}).SetLimit(limit)
	cur, err := r.collection.Find(ctx, bson.M{"user_id": userID}, opts)
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)

	var logs []GameLog
	if err := cur.All(ctx, &logs); err != nil {
		return nil, err
	}
	return logs, nil
}
