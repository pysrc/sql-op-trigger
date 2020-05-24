## 数据历史记录实现

Mysql数据操作历史记录备份

## example

### 原始表

```sql
CREATE TABLE `my_table` (
  `hid` bigint(20) NOT NULL AUTO_INCREMENT,
  `name` varchar(255) DEFAULT NULL,
  `indate` date DEFAULT NULL,
  `age` int(11) DEFAULT NULL,
  PRIMARY KEY (`hid`)
) ENGINE=InnoDB AUTO_INCREMENT=1 DEFAULT CHARSET=utf8mb4;
```

### 配置文件

config.json

```json
{
	"Ip": "192.168.137.4",
	"Port": "3306",
	"User": "root",
	"Password": "root",
	"Database": "my_test",
	"HisSuffix": "his",
	"Tables": ["my_table"]
}
```

### 生成历史记录表及触发器

```sql
-- my_table

CREATE TABLE my_table_his (
  `his_id` bigint(20) NOT NULL AUTO_INCREMENT,
  `his_type` varchar(255) DEFAULT NULL,
  `his_date` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `hid` bigint(20),
  `name` varchar(255),
  `indate` date,
  `age` int(11),
  PRIMARY KEY (`his_id`)
) ENGINE=InnoDB AUTO_INCREMENT=1 DEFAULT CHARSET=utf8mb4;

CREATE TRIGGER my_table_insert_tk AFTER INSERT
ON my_table FOR EACH ROW
BEGIN
    INSERT INTO my_table_his(`his_type`, `hid`, `name`, `indate`, `age`) VALUES ('insert', NEW.`hid`, NEW.`name`, NEW.`indate`, NEW.`age`);
END;

CREATE TRIGGER my_table_update_tk AFTER UPDATE
ON my_table FOR EACH ROW
BEGIN
    INSERT INTO my_table_his(`his_type`, `hid`, `name`, `indate`, `age`) VALUES ('update', NEW.`hid`, NEW.`name`, NEW.`indate`, NEW.`age`);
END;

CREATE TRIGGER my_table_delete_tk BEFORE DELETE
ON my_table FOR EACH ROW
BEGIN
    INSERT INTO my_table_his(`his_type`, `hid`, `name`, `indate`, `age`) VALUES ('delete', OLD.`hid`, OLD.`name`, OLD.`indate`, OLD.`age`);
END;

```

