const gp = require("geppetto");
const turn = gp.turn()
  .system("You inspect images.")
  .user(m => m.text("What is in this image?").imageURL("https://example.invalid/image.png"))
  .build();
const snapshot = turn.toJSON();
console.log(JSON.stringify({ blocks: snapshot.blocks.length, images: snapshot.blocks[1].payload.images.length }, null, 2));
