package basic

//go:generate events-gen -p basic -o events.go .

// event:"UserCreated"
// event:"UserRemoved"
type User struct {
	ID   int64
	Name string
}

// event:"UserSubscribed"
// event:"UserLeaved"
type Subscription struct {
	UserID int64
	Mail   string
}
