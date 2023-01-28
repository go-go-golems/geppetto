| Name | Type | Required | Example value | Description |
| --- | --- | --- | --- | --- |
| item_id | string | Yes* | SKU_12345 |  The ID of the item.  *One of item_id or item_name is required.   |
| item_name | string | Yes* | Stan and Friends Tee |  The name of the item.  *One of item_id or item_name is required.   |
| affiliation | string | No | Google Store |  A product affiliation to designate a supplying company or brick and mortar store location.
 Note: `affiliation` is only available at the item-scope.  |
| coupon | string | No | SUMMER_FUN | The coupon name/code associated with the item. Event-level and item-level coupon parameters are independent.  |
| discount | number | No | 2.22 | The monetary discount value associated with the item. |
| index | number | No | 5 | The index/position of the item in a list. |
| item_brand | string | No | Google | The brand of the item. |
| item_category | string | No | Apparel | The category of the item. If used as part of a category hierarchy or taxonomy then this will be the first category. |
| item_category2 | string | No | Adult | The second category hierarchy or additional taxonomy for the item. |
| item_category3 | string | No | Shirts | The third category hierarchy or additional taxonomy for the item. |
| item_category4 | string | No | Crew | The fourth category hierarchy or additional taxonomy for the item. |
| item_category5 | string | No | Short sleeve | The fifth category hierarchy or additional taxonomy for the item. |
|
| item_list_id | string | No | related_products | The ID of the list in which the item was presented to the user. If set, event-level item_list_id is ignored. If not set, event-level item_list_id is used, if present.  |
| item_list_name | string | No | Related products | The name of the list in which the item was presented to the user. If set, event-level item_list_name is ignored. If not set, event-level item_list_name is used, if present.  |
| item_variant | string | No | green | The item variant or unique code or description for additional item details/options. |
| location_id | string | No | ChIJIQBpAG2ahYAR_6128GcTUEo (the Google Place ID for San Francisco) |  The physical location associated with the item (e.g. the physical store location). It's recommended to use the Google Place ID that corresponds to the associated item. A custom location ID can also be used.
Note: `location id` is only available at the item-scope.  |
| price | number | No | 9.99 |  The monetary price of the item, in units of the specified currency parameter.  |
| quantity | number | No | 1 |  Item quantity. If not set, quantity is set to 1.  |