# guess-three
Guess three

## Previews

![Screenshot 1](1.png)

![Screenshot 2](2.png)


## Deploy
```bash
sudo cp /var/www/guess-three/guess-three.conf /etc/supervisor/conf.d/ && \
sudo mkdir -p /var/log/guess-three && \
sudo chown www-data:www-data /var/log/guess-three && \
sudo supervisorctl reread && \
sudo supervisorctl update && \
sudo supervisorctl start guess-three && \
sudo supervisorctl status
```

## Restart
```bash
sudo supervisorctl restart guess-three
```

