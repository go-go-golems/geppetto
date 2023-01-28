<!-- order:23 -->
## spend_virtual_currency

This event measures the sale of virtual goods in your app and helps you identify which virtual goods are the most popular.

### Parameters

| Name                    | Type     | Required | Example value | Description                                                  |
| ----------------------- | -------- | -------- | ------------- | ------------------------------------------------------------ |
| `value`                 | `number` | **Yes**  | 5             | The value of the virtual currency.                           |
| `virtual_currency_name` | `string` | **Yes**  | Gems          | The name of the virtual currency.                            |
| `item_name`             | `string` | No       | Starter Boost | The name of the item the virtual currency is being used for. |
