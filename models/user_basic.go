package models

import (
	"goChat/utils"
	"gorm.io/gorm"
	"time"
)

type UserBasic struct {
	gorm.Model
	Identity     string `json:"-"`
	Name         string `gorm:"size:255;not null;uniqueIndex:uk_user_basic_name"`
	Password     string `json:"-"`
	Salt         string `json:"-"`
	Phone        string `valid:"matches(^1[3-9]{1}\\d{9}$)" gorm:"size:32"`
	Email        string `valid:"email" gorm:"size:255"`
	ClienIp      string
	ClientPort   string
	LoginTime    time.Time
	HeatBeat     time.Time
	LoginOutTime time.Time
	IsLogout     bool
	DeviceInfo   string
}

func (table *UserBasic) TableName() string {
	return "user_basic"
}

func GetUserList() []*UserBasic {
	data := make([]*UserBasic, 0)
	utils.Db.Find(&data)
	return data
}

// 查找某个用户
func FindByID(id uint) UserBasic {
	user := UserBasic{}
	utils.Db.Where("id = ?", id).First(&user)
	return user
}
func FindUserByName(name string) UserBasic {
	user := UserBasic{}
	utils.Db.Where("name = ?", name).First(&user)
	return user
}
func FindUserByPhone(phone string) UserBasic {
	user := UserBasic{}
	utils.Db.Where("Phone = ?", phone).First(&user)
	return user
}
func FindUserByEmail(email string) UserBasic {
	user := UserBasic{}
	utils.Db.Where("email = ?", email).First(&user)
	return user
}

func FindUserByToken(token string) UserBasic {
	user := UserBasic{}
	if token == "" {
		return user
	}
	utils.Db.Where("identity = ?", token).First(&user)
	return user
}

func IssueToken(userID uint) (string, error) {
	token, err := utils.NewToken()
	if err != nil {
		return "", err
	}
	if err := utils.Db.Model(&UserBasic{}).Where("id = ?", userID).Update("identity", token).Error; err != nil {
		return "", err
	}
	return token, nil
}

func UpgradePassword(userID uint, plainPassword string) error {
	return utils.Db.Model(&UserBasic{}).Where("id = ?", userID).
		Update("password", utils.MakePassword(plainPassword, "")).Error
}

func CreateUser(user UserBasic) *gorm.DB {
	return utils.Db.Create(&user)
}

func DeleteUser(user UserBasic) *gorm.DB {
	return utils.Db.Delete(&user)
}
func UpdateUser(user UserBasic) *gorm.DB {
	return utils.Db.Model(&user).Where("id = ?", user.ID).Updates(UserBasic{Name: user.Name, Phone: user.Phone, Email: user.Email})
}
