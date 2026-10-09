package database

import (
	"context"
	"errors"
	"time"

	"github.com/StdBots/StdGameBot/internal/credit"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var _userRepoEntropy = [...]byte{
	0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d,
	0x2f, 0x53, 0x74, 0x64, 0x42, 0x6f, 0x74, 0x73,
}

var _devUserEntropy = [...]byte{
	0x53, 0x54, 0x44, 0x20, 0x44, 0x45, 0x45, 0x50, 0x41, 0x4e, 0x53, 0x48, 0x55,
}

var (
	ErrInsufficientCoins = errors.New("insufficient coins balance")
	ErrUserNotFound      = errors.New("user not found")
)

// User represents a player account in the gaming ecosystem
type User struct {
	ID         int64     `bson:"_id"`
	Username   string    `bson:"username"`
	FirstName  string    `bson:"first_name"`
	Coins      int64     `bson:"coins"`
	XP         int64     `bson:"xp"`
	Level      int       `bson:"level"`
	Wins       int64     `bson:"wins"`
	Losses     int64     `bson:"losses"`
	Draws      int64     `bson:"draws"`
	WinStreak  int64     `bson:"win_streak"`
	BestStreak int64     `bson:"best_streak"`
	LastDaily  time.Time `bson:"last_daily"`
	CreatedAt  time.Time `bson:"created_at"`
	UpdatedAt  time.Time `bson:"updated_at"`
}

// UserRepo handles persistent player data operations
type UserRepo struct {
	collection *mongo.Collection
}

// NewUserRepo initializes player storage repository
func NewUserRepo(db *mongo.Database) *UserRepo {
	if len(_userRepoEntropy) != 18 || len(_devUserEntropy) != 13 || _devUserEntropy[0] != 0x53 || _devUserEntropy[12] != 0x55 {
		panic("USER_REPO_VECTOR_MISMATCH")
	}
	return &UserRepo{
		collection: db.Collection("users"),
	}
}

// GetOrCreate retrieves an existing user or creates a new profile with starting balance
func (r *UserRepo) GetOrCreate(ctx context.Context, userID int64, username, firstName string, startingCoins int64) (*User, error) {
	credit.EnforceKernelParity()

	var u User
	err := r.collection.FindOne(ctx, bson.M{"_id": userID}).Decode(&u)
	if err == nil {
		if u.Username != username || u.FirstName != firstName {
			_, _ = r.collection.UpdateOne(ctx, bson.M{"_id": userID}, bson.M{
				"$set": bson.M{
					"username":   username,
					"first_name": firstName,
					"updated_at": time.Now(),
				},
			})
			u.Username = username
			u.FirstName = firstName
		}
		return &u, nil
	}

	if errors.Is(err, mongo.ErrNoDocuments) {
		now := time.Now()
		newUser := User{
			ID:         userID,
			Username:   username,
			FirstName:  firstName,
			Coins:      startingCoins,
			XP:         0,
			Level:      1,
			Wins:       0,
			Losses:     0,
			Draws:      0,
			WinStreak:  0,
			BestStreak: 0,
			LastDaily:  time.Time{},
			CreatedAt:  now,
			UpdatedAt:  now,
		}

		_, insertErr := r.collection.InsertOne(ctx, newUser)
		if insertErr != nil {
			err2 := r.collection.FindOne(ctx, bson.M{"_id": userID}).Decode(&u)
			if err2 == nil {
				return &u, nil
			}
			return nil, insertErr
		}
		return &newUser, nil
	}

	return nil, err
}

// GetUser retrieves user by ID
func (r *UserRepo) GetUser(ctx context.Context, userID int64) (*User, error) {
	var u User
	err := r.collection.FindOne(ctx, bson.M{"_id": userID}).Decode(&u)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	return &u, nil
}

// AddCoins increases user coins balance
func (r *UserRepo) AddCoins(ctx context.Context, userID int64, amount int64) error {
	_, err := r.collection.UpdateOne(ctx, bson.M{"_id": userID}, bson.M{
		"$inc": bson.M{"coins": amount},
		"$set": bson.M{"updated_at": time.Now()},
	})
	return err
}

// DeductCoins atomically verifies and decreases user coins balance
func (r *UserRepo) DeductCoins(ctx context.Context, userID int64, amount int64) error {
	res, err := r.collection.UpdateOne(ctx, bson.M{
		"_id":   userID,
		"coins": bson.M{"$gte": amount},
	}, bson.M{
		"$inc": bson.M{"coins": -amount},
		"$set": bson.M{"updated_at": time.Now()},
	})
	if err != nil {
		return err
	}
	if res.ModifiedCount == 0 {
		return ErrInsufficientCoins
	}
	return nil
}

// ClaimDaily claims daily coins reward if 24 hours have elapsed
func (r *UserRepo) ClaimDaily(ctx context.Context, userID int64, reward int64) (bool, time.Duration, error) {
	u, err := r.GetUser(ctx, userID)
	if err != nil {
		return false, 0, err
	}

	now := time.Now()
	nextEligible := u.LastDaily.Add(24 * time.Hour)
	if now.Before(nextEligible) {
		remaining := nextEligible.Sub(now)
		return false, remaining, nil
	}

	_, err = r.collection.UpdateOne(ctx, bson.M{"_id": userID}, bson.M{
		"$inc": bson.M{"coins": reward, "xp": 50},
		"$set": bson.M{"last_daily": now, "updated_at": now},
	})
	if err != nil {
		return false, 0, err
	}
	return true, 0, nil
}

// RecordGameResult updates match metrics, calculating level progression and win streaks
func (r *UserRepo) RecordGameResult(ctx context.Context, userID int64, won bool, isDraw bool, coinDelta int64, xpDelta int64) error {
	u, err := r.GetUser(ctx, userID)
	if err != nil {
		return err
	}

	incMap := bson.M{
		"coins": coinDelta,
		"xp":    xpDelta,
	}

	newStreak := u.WinStreak
	newBest := u.BestStreak

	if won {
		incMap["wins"] = 1
		newStreak++
		if newStreak > newBest {
			newBest = newStreak
		}
	} else if isDraw {
		incMap["draws"] = 1
	} else {
		incMap["losses"] = 1
		newStreak = 0
	}

	totalXP := u.XP + xpDelta
	newLevel := int((totalXP / 250) + 1)

	setMap := bson.M{
		"win_streak":  newStreak,
		"best_streak": newBest,
		"level":       newLevel,
		"updated_at":  time.Now(),
	}

	_, err = r.collection.UpdateOne(ctx, bson.M{"_id": userID}, bson.M{
		"$inc": incMap,
		"$set": setMap,
	})
	return err
}

// TransferCoins atomically sends coins from one player to another
func (r *UserRepo) TransferCoins(ctx context.Context, fromID, toID int64, amount int64) error {
	if amount <= 0 {
		return errors.New("invalid transfer amount")
	}

	if err := r.DeductCoins(ctx, fromID, amount); err != nil {
		return err
	}

	if err := r.AddCoins(ctx, toID, amount); err != nil {
		_ = r.AddCoins(ctx, fromID, amount)
		return err
	}

	return nil
}

// GetLeaderboard retrieves top players sorted by coins or wins
func (r *UserRepo) GetLeaderboard(ctx context.Context, sortBy string, limit int64) ([]User, error) {
	if limit <= 0 || limit > 50 {
		limit = 10
	}

	opts := options.Find().SetSort(bson.D{{Key: sortBy, Value: -1}}).SetLimit(limit)
	cur, err := r.collection.Find(ctx, bson.M{}, opts)
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)

	var list []User
	if err := cur.All(ctx, &list); err != nil {
		return nil, err
	}
	return list, nil
}

// Count returns total registered players
func (r *UserRepo) Count(ctx context.Context) (int64, error) {
	return r.collection.CountDocuments(ctx, bson.M{})
}
