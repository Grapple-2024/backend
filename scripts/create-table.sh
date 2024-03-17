## CREATE GYMS TABLE
aws dynamodb create-table \
   --endpoint-url http://localhost:8000 --region=local \
   --table-name grapple-gyms \
   --attribute-definitions AttributeName=pk,AttributeType=S AttributeName=creator,AttributeType=S \
   --key-schema AttributeName=pk,KeyType=HASH \
   --provisioned-throughput ReadCapacityUnits=5,WriteCapacityUnits=5 \
   --global-secondary-indexes "[{ \
       \"IndexName\": \"CreatorIndex\", \
       \"KeySchema\": [ \
           {\"AttributeName\":\"creator\",\"KeyType\":\"HASH\"} \
       ], \
       \"Projection\": { \
           \"ProjectionType\": \"ALL\" \
       }, \
       \"ProvisionedThroughput\": { \
           \"ReadCapacityUnits\": 5, \
           \"WriteCapacityUnits\": 5 \
       } \
   }]"


## CREATE GYM ANNOUNCEMENTS
aws dynamodb create-table \
   --endpoint-url http://localhost:8000 --region=local \
   --table-name grapple-gym-announcements \
   --attribute-definitions AttributeName=pk,AttributeType=S AttributeName=dummy,AttributeType=S AttributeName=updated_at,AttributeType=S AttributeName=gym_id,AttributeType=S \
   --key-schema AttributeName=pk,KeyType=HASH \
   --provisioned-throughput ReadCapacityUnits=20,WriteCapacityUnits=20 \
   --global-secondary-indexes "[{ \
       \"IndexName\": \"GymIndex\", \
       \"KeySchema\": [ \
           {\"AttributeName\":\"gym_id\",\"KeyType\":\"HASH\"} \
       ], \
       \"Projection\": { \
           \"ProjectionType\": \"ALL\" \
       }, \
       \"ProvisionedThroughput\": { \
           \"ReadCapacityUnits\": 20, \
           \"WriteCapacityUnits\": 20 \
       } \
   }, \
   { \
       \"IndexName\": \"LastUpdatedIndex\", \
       \"KeySchema\": [ \
           {\"AttributeName\":\"dummy\",\"KeyType\":\"HASH\"}, \
           {\"AttributeName\":\"updated_at\",\"KeyType\":\"RANGE\"} \
       ], \
       \"Projection\": { \
           \"ProjectionType\": \"ALL\" \
       }, \
       \"ProvisionedThroughput\": { \
           \"ReadCapacityUnits\": 20, \
           \"WriteCapacityUnits\": 20 \
       } \
   }]"


## CREATE GYM VIDEOS TABLE
aws dynamodb create-table \
   --endpoint-url http://localhost:8000 --region=local \
   --table-name grapple-gym-videos \
   --attribute-definitions AttributeName=pk,AttributeType=S AttributeName=gym_id,AttributeType=S\
   --key-schema AttributeName=pk,KeyType=HASH \
   --provisioned-throughput ReadCapacityUnits=5,WriteCapacityUnits=5 \
   --global-secondary-indexes "[{ \
       \"IndexName\": \"GymIndex\", \
       \"KeySchema\": [ \
           {\"AttributeName\":\"gym_id\",\"KeyType\":\"HASH\"} \
       ], \
       \"Projection\": { \
           \"ProjectionType\": \"ALL\" \
       }, \
       \"ProvisionedThroughput\": { \
           \"ReadCapacityUnits\": 5, \
           \"WriteCapacityUnits\": 5 \
       } \
   }]"


## CREATE GYM REQUESTS
aws dynamodb create-table \
   --endpoint-url http://localhost:8000 --region=local \
   --table-name grapple-gym-requests \
   --attribute-definitions AttributeName=pk,AttributeType=S AttributeName=dummy,AttributeType=S AttributeName=created_at,AttributeType=S \
   --key-schema AttributeName=pk,KeyType=HASH \
   --provisioned-throughput ReadCapacityUnits=50,WriteCapacityUnits=50 \
   --global-secondary-indexes "[{ \
       \"IndexName\": \"CreatedAtIndex\", \
       \"KeySchema\": [ \
           {\"AttributeName\":\"dummy\",\"KeyType\":\"HASH\"}, \
           {\"AttributeName\":\"created_at\",\"KeyType\":\"RANGE\"} \
       ], \
       \"Projection\": { \
           \"ProjectionType\": \"ALL\" \
       }, \
       \"ProvisionedThroughput\": { \
           \"ReadCapacityUnits\": 50, \
           \"WriteCapacityUnits\": 50 \
       } \
   }]"