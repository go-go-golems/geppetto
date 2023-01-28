<!-- order:13 -->
## post_score

Send this event when the user posts a score. Use this event to understand how users are performing in your game and correlate high scores with audiences or behaviors.

### Parameters

| Name        | Type     | Required | Example value | Description                            |
| ----------- | -------- | -------- | ------------- | -------------------------------------- |
| `score`     | `number` | **Yes**  | 10000         | The score to post.                     |
| `level`     | `number` | No       | 5             | The level for the score.               |
| `character` | `string` | No       | Player 1      | The character that achieved the score. |
