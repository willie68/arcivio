# automatische Übersetzungen mit libretrans

docker run -d -p 5000:5000 -e LT_LOAD_ONLY="en,de" -v libretranslate-data:/home/libretranslate/.local  --name libretranslate --restart unless-stopped libretranslate/libretranslate

kann auch per api-key geschlossen werden.