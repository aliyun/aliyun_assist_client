package taskerrors

type Stringer string

func (s Stringer) String() string {
	return string(s)
}
