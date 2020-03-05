package params

func (bs *Basic) Add(a, b int) {}

type Ooops struct {
}

func (Basic) OneMore(oop Ooops) {}

func Mutate(a, b string) {}
