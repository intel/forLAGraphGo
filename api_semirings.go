package forLAGraphGo

import GrB "github.com/intel/forGraphBLASGo"

func PlusFirst[T GrB.Number, Din2 any]() (addition GrB.Monoid[T], multiplication GrB.BinaryOp[T, T, Din2], identity T) {
	return GrB.PlusMonoid[T], GrB.First[T, Din2], 0
}

func PlusSecond[T GrB.Number, Din1 any]() (addition GrB.Monoid[T], multiplication GrB.BinaryOp[T, Din1, T], identity T) {
	return GrB.PlusMonoid[T], GrB.Second[Din1, T], 0
}

func PlusOne[T GrB.Number, Din1, Din2 any]() (addition GrB.Monoid[T], multiplication GrB.BinaryOp[T, Din1, Din2], identity T) {
	return GrB.PlusMonoid[T], GrB.Oneb[T, Din1, Din2], 0
}

func Structural[T GrB.Number, Din1, Din2 any]() (addition GrB.Monoid[T], multiplication GrB.BinaryOp[T, Din1, Din2], identity T) {
	return GrB.MinMonoid[T], GrB.Oneb[T, Din1, Din2], 0
}

func StructuralBool[Din1, Din2 any]() (addition GrB.Monoid[bool], multiplication GrB.BinaryOp[bool, Din1, Din2], identity bool) {
	return GrB.LOrMonoid, GrB.Trueb[Din1, Din2], false
}
