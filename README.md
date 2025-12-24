# Logistic Telegram Bot 

This project is created to be a partner and help for both logistic-managers and drivers working at V&R Spedition.

Currently, I am cleaning parts of it, so it has more clear code, as well as less bugs and more inpenetrability.

All of the code is licensed under BSD-3 license.

# Building

To build and use this code, you should do the following commands:

```bash
git clone https://github.com/appleofeden110/logistic_tbot

go mod tidy

go build

./logistictbot
```

You should provide your own TELEGRAM_API, as well.

# DB

This bot uses sqlite3 for easiness, and should give you easy time when you simply create a db with name "bot.db" in the root of the project.

It checks every needed table and creates it as necessary by the code. 


# Contact

You can write me on [nazarkaniuka6@gmail.com](mailto:nazarkaniuka6@gmail.com), if you have any questions.
