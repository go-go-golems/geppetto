## add_to_wishlist

The event signifies that an item was added to a wishlist. Use this event to identify popular gift items in your app.

### Parameters





| Name       | Type          | Required  | Example value | Description                                                                                                                                                                                                                                                                   |
| ---------- | ------------- | --------- | ------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `currency` | `string`      | **Yes\*** | USD           | Currency of the items associated with the event, in [3-letter ISO 4217](https://en.wikipedia.org/wiki/ISO_4217#Active_codes) format. \* If you set `value` then `currency` is required for revenue metrics to be computed accurately.                                         |
| `value`    | `number`      | **Yes\*** | 7.77          | The monetary value of the event. \* `value` is typically required for meaningful reporting. If you [mark the event as a conversion](https://support.google.com/analytics/answer/9267568) then it's recommended you set `value`. \* `currency` is required if you set `value`. |
| `items`    | `Array<Item>` | **Yes**   |               | The items for the event.                                                                                                                                                                                                                                                      |
