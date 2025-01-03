package kickvmhandle

func (h *VerifyHandle) DoAction() error{
	return nil
}

func (h *VerifyHandle) CheckAction() bool{
	return true
}

type VerifyHandle struct {
	code string
}

func NewVerifyHandle(code string) *VerifyHandle{
	return  &VerifyHandle{
		code:code,
	}
}
