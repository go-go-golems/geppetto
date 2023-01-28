#!/usr/bin/env bash

# Convert camel_case_name to CamelCaseName
function convert_to_camelcase() {
    echo $1 | perl -pe 's/(^|_)./uc($&)/ge;s/_//g'
}

DIR=examples/gtmgen

cat examples/gtmgen/events2.txt | while read i; do
  FILENAME=$(basename "$i")
  PHPFILENAME=${FILENAME%.*}.php
  # strip XXX- number in front
  PHPFILENAME=${PHPFILENAME#*-}
  # convert camel_case to CamelCase
  PHPFILENAME=$(convert_to_camelcase $PHPFILENAME)

  PHPOUTPUTDIR=$DIR/php
  if [ ! -d $PHPOUTPUTDIR ]; then
    mkdir $PHPOUTPUTDIR
  fi
  PHPOUTPUTFILE=$PHPOUTPUTDIR/$PHPFILENAME

  echo "Generating $PHPFILENAME"
  go run ./cmd/pinocchio \
      prompts gtmgen \
      --language php \
      --type "class" \
      --instructions "Comments on one line using //. Don't append Event to the class name. No getters, public attributes. add php8 constructor. no constructor comment." \
      "$i" | tee $PHPOUTPUTFILE

  TYPESCRIPTOUTPUTDIR=$DIR/typescript
  if [ ! -d $TYPESCRIPTOUTPUTDIR ]; then
    mkdir $TYPESCRIPTOUTPUTDIR
  fi

  TYPESCRIPTFILENAME=${FILENAME%.*}.ts
  # strip XXX- number in front
  TYPESCRIPTFILENAME=${TYPESCRIPTFILENAME#*-}
  # convert camel_case to CamelCase
  TYPESCRIPTFILENAME=$(convert_to_camelcase $TYPESCRIPTFILENAME)

  TYPESCRIPTOUTPUTFILE=$TYPESCRIPTOUTPUTDIR/$TYPESCRIPTFILENAME

  echo "Generating $TYPESCRIPTFILENAME"
  go run ./cmd/pinocchio \
      prompts gtmgen \
      --language typescript \
      --type "interface" \
      --instructions "Make optional attributes optional. Comments on one line using //. Don't append Event to the interface name." \
      "$i" | tee "$TYPESCRIPTOUTPUTFILE"
done