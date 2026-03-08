package notify

import "context"

func init() {
	RegisterSenderFactory(func(_ context.Context, bc BuildContext) ([]Sender, error) {
		s := bc.Config.buildEmailSender()
		if s == nil {
			return nil, nil
		}
		return []Sender{s}, nil
	})
}
