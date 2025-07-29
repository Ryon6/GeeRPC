package Test

import (
	"errors"
)

type CalcService struct{}

func (s *CalcService) Add(args *CArgs, reply *int) error {
	*reply = args.A + args.B
	return nil
}

func (s *CalcService) Sub(args *CArgs, reply *int) error {
	*reply = args.A - args.B
	return nil
}

func (s *CalcService) Mul(args *CArgs, reply *int) error {
	*reply = args.A * args.B
	return nil
}

func (s *CalcService) Div(args *CArgs, reply *int) error {
	if args.B == 0 {
		return errors.New("division by zero")
	}
	*reply = args.A / args.B
	return nil
}

type CArgs struct {
	A, B int
}
