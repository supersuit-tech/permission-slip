package mobilepush

import (
	"context"
	"log"

	"github.com/supersuit-tech/permission-slip/notify"
)

func init() {
	notify.RegisterSenderFactory("mobile-push", func(_ context.Context, bc notify.BuildContext) ([]notify.Sender, error) {
		if bc.DB == nil {
			return nil, nil
		}

		if bc.Config.ExpoAccessToken != "" {
			log.Println("Mobile Push: Expo access token configured (authenticated mode)")
		} else {
			log.Println("Mobile Push: no EXPO_ACCESS_TOKEN set, using unauthenticated mode (lower rate limits)")
		}

		return []notify.Sender{New(bc.DB, bc.Config.ExpoAccessToken)}, nil
	})
}
