aws dynamodb query \
   --endpoint-url http://localhost:8000 --region=local \
   --table-name grapple-gym-videos \
   --key-condition-expression "pk = :pk" \
   --filter-expression "contains(disciplines, :discipline)" \
   --expression-attribute-values '{":pk": {"S": "Z3ltVmlkZW8jLy8tNjIxMzU1OTY4MDA="}, ":discipline":{"S": "mmaa"}}'
