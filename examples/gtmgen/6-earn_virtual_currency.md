<!-- order:6 -->
## earn_virtual_currency

This event measures the awarding of virtual currency. Log this along with [spend_virtual_currency](#spend_virtual_currency) to better understand your virtual economy.

### Parameters

| Name                    | Type     | Required | Example value | Description                        |
| ----------------------- | -------- | -------- | ------------- | ---------------------------------- |
| `virtual_currency_name` | `string` | No       | Gems          | The name of the virtual currency.  |
| `value`                 | `number` | No       | 5             | The value of the virtual currency. |
