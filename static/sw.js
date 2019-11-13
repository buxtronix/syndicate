self.addEventListener('push', function(event) {
      console.log(`[Service Worker] Push had this data: "${event.data.text()}"`);

      var message = JSON.parse(event.data.text());
      self.click_url = message.URI;
      const title = 'Netops Beer Syndicate';
      const options = {
              body: message.Message,
              icon: '/static/beer-icon.png',
              badge: '/static/beer-badge.png'
            };

      event.waitUntil(self.registration.showNotification(title, options));
});


self.addEventListener('notificationclick', function(event) {
      console.log('[Service Worker] Notification click Received.');

      event.notification.close();

      event.waitUntil(
              clients.openWindow(self.click_url)
            );
});
