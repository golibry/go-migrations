## Examples  
  
Each folder integrates with a storage type (repository) the library supports (mysql, mongo, postgres).  
To play with these examples, follow the bellow steps:  
  
1. Change directory to project root
2. Start the containers: ``docker compose up -d``
3. SSH into lib-dev container: ``docker exec -it lib-dev bash``
4. Change directory to one of the available storage integrations: ``cd ./_examples/mysql`` (or ``./_examples/mongo`` or ``./_examples/postgres``)
5. Build the binary with the proper build tag:  
   - MySQL: ``go build -tags mysql -o ./bin/migrate``  
   - MongoDB: ``go build -tags mongo -o ./bin/migrate``  
   - Postgres: ``go build -tags postgres -o ./bin/migrate``  
6. Create a database with the name used in your connection settings (see .env file)  
7. Get helpful info from the "migrate" binary: ``./bin/migrate help``
8. Run one migration Up() from the "migrate" binary: ``./bin/migrate up``
9. Run migrations (3) Up() from the "migrate" binary: ``./bin/migrate up --steps=3``
10. Run all migrations Up() from the "migrate" binary: ``./bin/migrate up --steps=all``