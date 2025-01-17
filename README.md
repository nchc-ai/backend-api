
```bash
$ docker run --name MySQL  -d -p 3306:3306 -e MYSQL_ROOT_PASSWORD=password nchc-ai/mysql:v2020.10 --default-authentication-plugin=mysql_native_password
```

```mysql
CREATE DATABASE nchc;
CREATE USER 'nchc'@'localhost' IDENTIFIED BY 'nchc';
CREATE USER 'nchc'@'%' IDENTIFIED BY 'nchc';
GRANT ALL ON nchc.* TO 'nchc'@'localhost';
GRANT ALL ON nchc.* TO 'nchc'@'%';
```

GO Oauth sample user
```mysql
use nchc;
insert into user (user, provider, role, repository) values ( "user@teacher", "go-oauth:test-provider", "teacher", "user");
insert into user (user, provider, role, repository) values ( "user@student", "go-oauth:test-provider", "student", "");
insert into user (user, provider, role, repository) values ( "user@admin", "go-oauth:test-provider", "superuser", "");
```

Google Oauth sample user
```mysql
use nchc;
insert into user (user, provider, role, repository) values ( "user@gmail.com", "google-oauth:google-provider", "teacher", "user");
insert into user (user, provider, role, repository) values ( "user.xxxx@gmail.com", "google-oauth:google-provider", "student", "");
insert into user (user, provider, role, repository) values ( "user.yyyy@gmail.com", "google-oauth:google-provider", "superuser", "");
```

Github oauth sample user
```mysql
use nchc;
insert into user (user, provider, role, repository) values ( "user", "github-oauth:github-provider", "teacher", "user");
```