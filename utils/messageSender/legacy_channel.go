package messageSender

import (
	"context"

	"github.com/komari-monitor/komari/utils/messageSender/factory"
)

// legacyChannel adapts a pre-rewrite message sender to the notification
// channel registry. Configuration is decoded on each send, so saved provider
// rows keep working without a process-global current provider.
type legacyChannel struct {
	ctor factory.MessageSenderConstructor
}

func (c *legacyChannel) Unload() error { return nil }

func (c *legacyChannel) Send(_ context.Context, notification Notification, config map[string]any) error {
	sender := c.ctor()
	if err := DecodeConfiguration(config, sender.GetConfiguration()); err != nil {
		return err
	}
	if err := sender.Init(); err != nil {
		return err
	}
	defer sender.Destroy()
	if eventSender, ok := sender.(factory.IEventMessageSender); ok {
		return eventSender.SendEvent(notification.Event)
	}
	return sender.SendTextMessage(notification.Message, notification.Title)
}

// RegisterLegacyChannels publishes the retained built-in providers
// (telegram, email, bark, and the other saved channels) on the new registry.
func RegisterLegacyChannels() error {
	for name, proto := range factory.GetAllMessageSenders() {
		if NotificationChannelRegistered(name) {
			continue
		}
		ctor, ok := factory.GetConstructor(name)
		if !ok || proto == nil {
			continue
		}
		if err := RegisterNotificationChannel(name, ManagedConfiguration(name, proto.GetConfiguration()), &legacyChannel{ctor: ctor}); err != nil {
			return err
		}
	}
	return nil
}
