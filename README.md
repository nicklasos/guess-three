# guess-three
Guess three

## Previews

![Screenshot 1](1.png)

![Screenshot 2](2.png)


## Deploy
```bash
sudo cp /var/www/guess/guess-three.conf /etc/supervisor/conf.d/ && \
sudo mkdir -p /var/log/guess && \
sudo chown www-data:www-data /var/log/guess && \
sudo supervisorctl reread && \
sudo supervisorctl update && \
sudo supervisorctl start guess-three
```

