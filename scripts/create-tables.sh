## CREATE GYMS TABLE
aws dynamodb create-table \
   --endpoint-url http://localhost:8000 --region=local \
   --table-name grapple-local-gyms \
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
   --table-name grapple-local-gym-announcements \
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

## CREATE GYM VIDEO SERIES TABLE
aws dynamodb create-table \
   --endpoint-url http://localhost:8000 --region=local \
   --table-name grapple-local-gym-video-series \
   --attribute-definitions AttributeName=pk,AttributeType=S AttributeName=updated_at,AttributeType=S AttributeName=dummy,AttributeType=S \
   --key-schema AttributeName=pk,KeyType=HASH \
   --provisioned-throughput ReadCapacityUnits=5,WriteCapacityUnits=5 \
   --global-secondary-indexes "[
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
   }
]"

## CREATE GYM VIDEOS TABLE
aws dynamodb create-table \
   --endpoint-url http://localhost:8000 --region=local \
   --table-name grapple-local-gym-videos \
   --attribute-definitions AttributeName=pk,AttributeType=S AttributeName=series_id,AttributeType=S AttributeName=updated_at,AttributeType=S AttributeName=dummy,AttributeType=S \
   --key-schema AttributeName=pk,KeyType=HASH \
   --provisioned-throughput ReadCapacityUnits=5,WriteCapacityUnits=5 \
   --global-secondary-indexes "[
   { \
       \"IndexName\": \"SeriesIndex\", \
       \"KeySchema\": [ \
           {\"AttributeName\":\"series_id\",\"KeyType\":\"HASH\"} \
       ], \
       \"Projection\": { \
           \"ProjectionType\": \"ALL\" \
       }, \
       \"ProvisionedThroughput\": { \
           \"ReadCapacityUnits\": 5, \
           \"WriteCapacityUnits\": 5 \
       } \
    }, 
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
   }
]"


## CREATE GYM REQUESTS
aws dynamodb create-table \
   --endpoint-url http://localhost:8000 --region=local \
   --table-name grapple-local-gym-requests \
   --attribute-definitions AttributeName=pk,AttributeType=S AttributeName=dummy,AttributeType=S AttributeName=created_at,AttributeType=S AttributeName=requestor_id,AttributeType=S AttributeName=gym_id,AttributeType=S \
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
   }, { \
       \"IndexName\": \"RequestorIndex\", \
       \"KeySchema\": [ \
           {\"AttributeName\":\"requestor_id\",\"KeyType\":\"HASH\"} \
       ], \
       \"Projection\": { \
           \"ProjectionType\": \"ALL\" \
       }, \
       \"ProvisionedThroughput\": { \
           \"ReadCapacityUnits\": 50, \
           \"WriteCapacityUnits\": 50 \
       } \
   }, { \
       \"IndexName\": \"GymIndex\", \
       \"KeySchema\": [ \
           {\"AttributeName\":\"gym_id\",\"KeyType\":\"HASH\"} \
       ], \
       \"Projection\": { \
           \"ProjectionType\": \"ALL\" \
       }, \
       \"ProvisionedThroughput\": { \
           \"ReadCapacityUnits\": 50, \
           \"WriteCapacityUnits\": 50 \
       } \
   }]"

## CREATE Emails
aws dynamodb create-table \
   --endpoint-url http://localhost:8000 --region=local \
   --table-name grapple-local-emails \
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

## CREATE user assets
aws dynamodb create-table \
   --endpoint-url http://localhost:8000 --region=local \
   --table-name grapple-local-user-assets \
   --attribute-definitions AttributeName=url,AttributeType=S AttributeName=user_id,AttributeType=S\
   --key-schema AttributeName=url,KeyType=HASH \
   --provisioned-throughput ReadCapacityUnits=50,WriteCapacityUnits=50 \
   --global-secondary-indexes "[{ \
    \"IndexName\": \"UserIndex\", \
    \"KeySchema\": [ \
        {\"AttributeName\":\"user_id\",\"KeyType\":\"HASH\"} \
    ], \
    \"Projection\": { \
        \"ProjectionType\": \"ALL\" \
    }, \
    \"ProvisionedThroughput\": { \
        \"ReadCapacityUnits\": 5, \
        \"WriteCapacityUnits\": 5 \
    } \
}]"

## CREATE user profiles
aws dynamodb create-table \
   --endpoint-url http://localhost:8000 --region=local \
   --table-name grapple-local-user-profiles \
   --attribute-definitions AttributeName=user_id,AttributeType=S \
   --key-schema AttributeName=user_id,KeyType=HASH \
   --provisioned-throughput ReadCapacityUnits=50,WriteCapacityUnits=50

