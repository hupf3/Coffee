package services

import (
	"errors"

	"github.com/XMatrixStudio/Coffee/App/models"
	"github.com/XMatrixStudio/Violet.SDK.Go"
)

// UserService 用户服务层
type UserService interface {
	Login(name, password string) (valid bool, data string, err error)
	Register(name, email, password string) error
	GetEmailCode(email string) error
	ValidEmail(email, vCode string) error

	InitViolet(c violetSdk.Config)
	GetLoginURL(backURL string) (url, state string)
	LoginByCode(code string) (userID string, err error)
	GetUserInfo(id string) (user models.User, err error)
	GetUserBaseInfo(id string) (user UserBaseInfo)
	UpdateUserInfo(id string) error
	UpdateUserName(id, name string) error

	AddFiles(id string, size int64) error
}

type userService struct {
	Violet   violetSdk.Violet
	Model    *models.UserModel
	UserInfo map[string]UserBaseInfo
	Service  *Service
}

// Login ...
func (s *userService) Login(name, password string) (valid bool, data string, err error) {
	res, err := s.Violet.Login(name, password)
	if err != nil {
		return
	}
	valid = res.Valid
	// 未激活邮箱
	if !valid {
		data = res.Email
		return
	}
	// 登陆成功
	data = res.Code
	return
}

func (s *userService) Register(name, email, password string) error {
	return s.Violet.Register(name, email, password)
}

func (s *userService) GetEmailCode(email string) error {
	return s.Violet.GetEmailCode(email)
}

func (s *userService) ValidEmail(email, vCode string) error {
	return s.Violet.ValidEmail(email, vCode)
}

func (s *userService) SetUserInfo(id string, info models.UserInfo) error {
	if info.NikeName == "new_user" {
		return errors.New("not_allow")
	}
	users, err := s.Model.GetUsers()
	if err != nil {
		return errors.New("not_allow")
	}
	for _, user := range users {
		if user.Info.NikeName == info.NikeName {
			return errors.New("not_allow")
		}
	}
	s.GetUserBaseInfo(id)
	s.UserInfo[id] = UserBaseInfo{
		Avatar: info.Avatar,
		Name:   info.NikeName,
		Gender: info.Gender,
	}
	return s.Model.SetUserInfo(id, info)
}

func (s *userService) InitViolet(c violetSdk.Config) {
	s.Violet = violetSdk.NewViolet(c)
}

func (s *userService) GetLoginURL(backURL string) (url, state string) {
	url, state = s.Violet.GetLoginURL(backURL)
	return
}

func (s *userService) LoginByCode(code string) (userID string, err error) {
	// 获取用户Token
	res, err := s.Violet.GetToken(code)
	if err != nil {
		return
	}
	// 保存数据并获取用户信息
	user, err := s.Model.GetUserByVID(res.UserID)
	if err == nil { // 数据库已存在该用户
		userID = user.ID.Hex()
		err = s.Model.SetUserToken(user.ID.Hex(), res.Token)
	} else if err.Error() == "not found" { // 数据库不存在此用户
		userNew, errN := s.Violet.GetUserBaseInfo(res.UserID, res.Token)
		if errN != nil {
			err = errN
			return
		}
		userBsonID, errN := s.Model.AddUser(res.UserID, res.Token, userNew.Email, userNew.Name, userNew.Info.Avatar, userNew.Info.Bio, userNew.Info.Gender)
		err = errN
		userID = userBsonID.Hex()
	}
	return
}

func (s *userService) GetUserInfo(id string) (user models.User, err error) {
	user, err = s.Model.GetUserByID(id)
	return
}

// UserBaseInfo 用户个性信息
type UserBaseInfo struct {
	Name   string
	Avatar string
	Gender int
}

// GetUserBaseInfo 从缓存中读取用户基本信息，如果不存在则从数据库中读取
func (s *userService) GetUserBaseInfo(id string) (user UserBaseInfo) {
	user, ok := s.UserInfo[id]
	if !ok {
		userInfo, err := s.GetUserInfo(id)
		if err != nil {
			return UserBaseInfo{
				Name:   "匿名用户",
				Avatar: "https://pic3.zhimg.com/50/v2-e2361d82ce7465808260f87bed4a32d0_im.jpg",
			}
		}
		user = UserBaseInfo{
			Name:   userInfo.Info.Name,
			Avatar: userInfo.Info.Avatar,
		}
		s.UserInfo[id] = user
	}
	return
}

func (s *userService) UpdateUserInfo(id string) error {
	user, err := s.GetUserInfo(id)
	if err != nil {
		return err
	}
	userInfo, err := s.Violet.GetUserBaseInfo(user.VioletID.Hex(), user.Token)
	if err != nil {
		return err
	}
	s.UserInfo[id] = UserBaseInfo{
		Avatar: userInfo.Info.Avatar,
		Name:   user.Info.Name,
	}
	return s.Model.SetUserInfo(id, models.UserInfo{
		Name:   user.Info.Name,
		Avatar: userInfo.Info.Avatar,
		Bio:    userInfo.Info.Bio,
		Gender: userInfo.Info.Gender,
	})
}

func (s *userService) UpdateUserName(id, name string) error {
	err := s.Model.SetUserName(id, name)
	if err != nil {
		return err
	}
	info := s.GetUserBaseInfo(id)
	s.UserInfo[id] = UserBaseInfo{
		Avatar: info.Avatar,
		Name:   name,
	}
	return nil
}

func (s *userService) AddFiles(id string, size int64) error {
	user, err := s.Model.GetUserByID(id)
	if err != nil {
		return err
	}
	// 容量超過限制
	if user.UsedSize+size > user.MaxSize {
		return errors.New("max_size")
	}
	return s.Model.SetCount(id, models.UsedSize, user.UsedSize+size)
}
