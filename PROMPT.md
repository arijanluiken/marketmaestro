Create an advanced crypto tradingbot in go using the actor model. Use https://github.com/anthdm/hollywood for the actor framework, make sure you understand the examples on how to implement the actors. Ensure the actors are seggregated and only use messages to communicate with each other.

The bot should support multiple exchanges so create an abstract interface first, then create a concrete implementation for bybit using https://github.com/hirokisan/bybit retrieve the exchange api keys from a .env file and store the rest of the configuration in yaml.

As for the capabilities/actors:

1. Supervisor actor

The supervisor actor is responsible for managing actors

2. Exchange actor

The exchange actor is responsible for interaction with the exchange, it should be retrieving klines and orderbook data via websocket. Use a factory of the abstract exchange interface to instantiate the exchange used in the exchange actor so more can be added later. The exchange actor is a child of the supervisor actor

3. Strategy actor

The strategy actor is a child of the exchange actor, it needs to subscribe on the kline and orderbook feed of the exchange actor to get the data for the strategies. The strategies should be implemented in starlark so ensure the strategy engine has lots of technical indicators and helper functions to get information about positions, orders etc. To make orders the strategy actor should interact with the order manager actor. Next to standard out logging using zerolog the api actor should also be able to retrieve realtime logs of the strategy

4. Order manager actor

The order manager actor is a child of the exchange actor it is responsible for order management and should also provide advanced orders like trailing stop loss orders.

5. Risk manager actor

The risk manager actor is a child of the exchange actor it is responsible for managing the risk of the running strategies

6. Portfolio actor

The portfolio actor is a child of the exchange actor it is responsible for providing portfolio information and tracking profit and loss figures

7. Api actor

The api actor is a child of the supervisor actor, it is responsible to expose a rest api using chi as the router. Ensure the api actor has an openapi spec and a route to retrieve the spec. The api actor should be able to interact with multiple exchange actors. do ensure the actor is seggregated from the other actors and only use messages between actors it needs information from. Next to a rest api the api actor should also provide a websocket transport for realtime updates to the ui actor

8. UI Actor

The ui actor is a child of the supervisor actor, it provides a webinterface for the trading bot. The UI actor serves the web pages and javascript the pages served will use the rest api and websocket of the api actor. Embed the assets for the ui directly into the binary using go's embed functionality. Use pure CSS as a framework to give the bot a modern look. Use a light theme for the bot.

9. Settings actor

The settings actor is a child of the exchange actor and is responsbile to persist config settings for actors that need them for example. 


For actors that need persistance use sqlite3 and golang-migrate for schema migrations. Ensure the schema migration files are also embedded directly in the binary using go's embed functionality. Do not store klines or orderbook information in the database use the exchange api's for that.