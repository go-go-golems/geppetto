## add_payment_info

This event signifies a user has submitted their payment information.

### Parameters

| Name           | Type          | Required  | Example value | Description                                                                                                                                                                                                                                                                   |
| -------------- | ------------- | --------- | ------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `currency`     | `string`      | **Yes\*** | USD           | Currency of the items associated with the event, in [3-letter ISO 4217](https://en.wikipedia.org/wiki/ISO_4217#Active_codes) format. \* If you set `value` then `currency` is required for revenue metrics to be computed accurately.                                         |
| `value`        | `number`      | **Yes\*** | 7.77          | The monetary value of the event. \* `value` is typically required for meaningful reporting. If you [mark the event as a conversion](https://support.google.com/analytics/answer/9267568) then it's recommended you set `value`. \* `currency` is required if you set `value`. |
| `coupon`       | `string`      | No        | SUMMER_FUN   | The coupon name/code associated with the event. Event-level and item-level `coupon` parameters are independent.                                                                                                                                                               |
| `payment_type` | `string`      | No        | Credit Card   | The chosen method of payment.                                                                                                                                                                                                                                                 |
| `items`        | `Array<Item>` | **Yes**   |               | The items for the event.                                                                                                                                                                                                                                                      |
