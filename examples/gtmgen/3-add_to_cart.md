<!-- order:3 -->
## add_to_cart

This event signifies that an item was added to a cart for purchase.

### Parameters

| Name       | Type          | Required  |  Description                                                                                                                                                                                                                                                                   |
| ---------- | ------------- | --------- |  ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `currency` | `string`      | **Yes\*** |  Currency of the items associated with the event, in [3-letter ISO 4217](https://en.wikipedia.org/wiki/ISO_4217#Active_codes) format. \* If you set `value` then `currency` is required for revenue metrics to be computed accurately.                                         |
| `value`    | `number`      | **Yes\*** |  The monetary value of the event. \* `value` is typically required for meaningful reporting. If you [mark the event as a conversion](https://support.google.com/analytics/answer/9267568) then it's recommended you set `value`. \* `currency` is required if you set `value`. |
| `items`    | `Array<Item>` | **Yes**   |  The items for the event.                                                                                                                                                                                                                                                      |

#### Item parameters

| Name             | Type     | Required  |  Description                                                                                                                                                                                                                                                                                                                  |
| ---------------- | -------- | --------- |  ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `item_id`        | `string` | **Yes\*** |  The ID of the item.\*One of `item_id` or `item_name` is required.                                                                                                                                                                                                                                                            |
| `item_name`      | `string` | **Yes\*** |  The name of the item.\*One of `item_id` or `item_name` is required.                                                                                                                                                                                                                                                          |
| `affiliation`    | `string` | No        |  A product affiliation to designate a supplying company or brick and mortar store location. Note: \`affiliation\` is only available at the item-scope.                                                                                                                                                                        |
| `coupon`         | `string` | No        |  The coupon name/code associated with the item. Event-level and item-level `coupon` parameters are independent.                                                                                                                                                                                                               |
| `discount`       | `number` | No        |  The monetary discount value associated with the item.                                                                                                                                                                                                                                                                        |
| `index`          | `number` | No        |  The index/position of the item in a list.                                                                                                                                                                                                                                                                                    |
| `item_brand`     | `string` | No        |  The brand of the item.                                                                                                                                                                                                                                                                                                       |
| `item_category`  | `string` | No        |  The category of the item. If used as part of a category hierarchy or taxonomy then this will be the first category.                                                                                                                                                                                                          |
| `item_category2` | `string` | No        |  The second category hierarchy or additional taxonomy for the item.                                                                                                                                                                                                                                                           |
| `item_category3` | `string` | No        |  The third category hierarchy or additional taxonomy for the item.                                                                                                                                                                                                                                                            |
| `item_category4` | `string` | No        |  The fourth category hierarchy or additional taxonomy for the item.                                                                                                                                                                                                                                                           |
| `item_category5` | `string` | No        |  The fifth category hierarchy or additional taxonomy for the item.                                                                                                                                                                                                                                                            |
|                  |          |           |                                                                                                                                                                                                                                                                                                                               |
| `item_list_id`   | `string` | No        |  The ID of the list in which the item was presented to the user. If set, event-level `item_list_id` is ignored. If not set, event-level `item_list_id` is used, if present.                                                                                                                                                   |
| `item_list_name` | `string` | No        |  The name of the list in which the item was presented to the user. If set, event-level `item_list_name` is ignored. If not set, event-level `item_list_name` is used, if present.                                                                                                                                             |
| `item_variant`   | `string` | No        |  The item variant or unique code or description for additional item details/options.                                                                                                                                                                                                                                          |
| `location_id`    | `string` | No        |  The physical location associated with the item (e.g. the physical store location). It's recommended to use the [Google Place ID](/maps/documentation/places/web-service/place-id) that corresponds to the associated item. A custom location ID can also be used. Note: \`location id\` is only available at the item-scope. |
| `price`          | `number` | No        |  The monetary price of the item, in units of the specified currency parameter.                                                                                                                                                                                                                                                |
| `quantity`       | `number` | No        |  Item quantity. If not set, `quantity` is set to 1.                                                                                                                                                                                                                                                                           |
