package login_sdk_go

import (
	"context"
	"errors"
	"fmt"
	"github.com/dgrijalva/jwt-go"
	"gitlab.loc/sdk-login/login-sdk-go/interfaces"
	"gitlab.loc/sdk-login/login-sdk-go/session"
	"google.golang.org/grpc"
	"log"
)

type Validator interface {
	Validate(jwt string) error
}

type SessionValidator struct {
	invalidationServiceClient session.InvalidationServiceClient
}

func NewSessionValidator(host string, port int) (*SessionValidator, error) {
	conn, err := grpc.Dial(fmt.Sprintf("%s:%d", host, port))

	if err != nil {
		return nil, fmt.Errorf("failed connect to grpc: %v", err)
	}

	return &SessionValidator{
		session.NewInvalidationServiceClient(conn),
	}, nil
}

type MasterValidator struct {
	Config
	rs256SigningKey  SigningKeyGetter
	hs256SigningKey  SigningKeyGetter
	sessionValidator *SessionValidator
}

func NewMasterValidator(config Config, client *interfaces.LoginApi) (*MasterValidator, error) {
	cks := NewCachedValidationKeysStorage(*client, config.Cache)
	rsa := RSAPublicKeyGetter{storage: cks, projectId: config.LoginProjectId}

	sessionValidator, err := NewSessionValidator(config.SessionApiHost, config.SessionApiPort)

	if err != nil {
		log.Printf("failed create session validator: %v", err)
	}

	return &MasterValidator{
		Config:           config,
		rs256SigningKey:  RS256SigningKeyGetter{config, rsa},
		hs256SigningKey:  HS256SigningKeyGetter{config.ShaSecretKey},
		sessionValidator: sessionValidator,
	}, nil
}

func (mv MasterValidator) Validate(tokenString string) error {
	parsedToken, err := jwt.ParseWithClaims(tokenString, &CustomClaims{}, func(token *jwt.Token) (interface{}, error) {
		signingMethod := token.Method
		switch signingMethod {
		case jwt.SigningMethodRS256:
			return mv.rs256SigningKey.getKey(token)
		case jwt.SigningMethodHS256:
			return mv.hs256SigningKey.getKey(token)
		default:
			return nil, errors.New(signingMethod.Alg() + " algorithm is not supported")
		}
	})

	if err == nil && !mv.Config.SkipSessionValidation && mv.sessionValidator != nil {
		return mv.sessionValidator.Validate(parsedToken)
	}

	return err
}

func (s SessionValidator) Validate(token *jwt.Token) error {
	claims, ok := token.Claims.(*CustomClaims)

	if !ok {
		log.Printf("type assertion error: CustomClaims")
		return nil
	}

	request := session.Request{
		Jti: claims.Id,
	}

	response, err := s.invalidationServiceClient.HaveBeenInvalidated(context.Background(), &request)

	if err != nil {
		log.Printf("InvalidationServiceClient exception: %s", err.Error())
		return nil
	}

	if response.Invalidated {
		return errors.New("token has been invalidated")
	}

	return nil
}
