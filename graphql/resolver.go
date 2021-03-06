//go:generate go run ../scripts/gqlgen.go -v
package graphql

import (
	"context"
	"log"
	"reflect"

	"github.com/99designs/gqlgen/graphql"
	"github.com/ProjectBlacktube/blacktube-graphql/manager"
	"github.com/ProjectBlacktube/blacktube-graphql/models"
	"github.com/mitchellh/mapstructure"
)

type Resolver struct {
	UserManager  *manager.UserQueryManager
	VideoManager *manager.VideoQueryManager
}

func (r *Resolver) Mutation() MutationResolver {
	return &mutationResolver{r}
}
func (r *Resolver) Query() QueryResolver {
	return &queryResolver{r}
}
func (r *Resolver) Video() VideoResolver {
	return &videoResolver{r}
}

type mutationResolver struct{ *Resolver }

func (r *mutationResolver) CreateUser(ctx context.Context, input models.NewUser) (*models.User, error) {
	return r.UserManager.NewUser(&input)
}
func (r *mutationResolver) UpdateUser(ctx context.Context, id string, mutation map[string]interface{}) (*models.User, error) {
	u, err := r.UserManager.FindUser(id)
	if err != nil {
		log.Panic(err)
	}

	err = applyMap(mutation, &u)
	if err != nil {
		return &models.User{}, err
	}

	return r.UserManager.UpdateUser(u)
}
func (r *mutationResolver) DeleteUser(ctx context.Context, id string) (*models.User, error) {
	return r.UserManager.DeleteUser(id)
}

func (r *mutationResolver) CreateVideo(ctx context.Context, input models.NewVideo) (*models.VideoNested, error) {
	return r.VideoManager.NewVideo(&input)
}
func (r *mutationResolver) DeleteVideo(ctx context.Context, id string) (*models.VideoNested, error) {
	return r.VideoManager.DeleteVideo(id)
}
func (r *mutationResolver) UpdateVideo(ctx context.Context, id string, mutation map[string]interface{}) (*models.VideoNested, error) {
	vn, err := r.VideoManager.FindVideo(id)
	if err != nil {
		log.Panic(err)
	}

	v, err := r.VideoManager.FromNested(vn)
	if err != nil {
		return &models.VideoNested{}, err
	}

	err = applyMap(mutation, &v)
	if err != nil {
		return &models.VideoNested{}, err
	}

	return r.VideoManager.UpdateVideo(v)
}

type queryResolver struct{ *Resolver }

func (r *queryResolver) Users(ctx context.Context) ([]*models.User, error) {
	return r.UserManager.AllUsers()
}
func (r *queryResolver) Videos(ctx context.Context) ([]*models.VideoNested, error) {
	return r.VideoManager.AllVideos()
}
func (r *queryResolver) VideoByKey(ctx context.Context, key string) (*models.VideoNested, error) {
	return r.VideoManager.FindVideoByKey(key)
}

type videoResolver struct{ *Resolver }

func (r *videoResolver) CreatedAt(ctx context.Context, video *models.VideoNested) (int64, error) {
	return video.CreatedAt.Unix(), nil
}

func applyMap(changes map[string]interface{}, to interface{}) error {
	dec, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		ErrorUnused: true,
		TagName:     "json",
		Result:      to,
		ZeroFields:  true,
		// This is needed to get mapstructure to call the gqlgen unmarshaler func for custom scalars (eg Date)
		DecodeHook: func(a reflect.Type, b reflect.Type, v interface{}) (interface{}, error) {
			if reflect.PtrTo(b).Implements(reflect.TypeOf((*graphql.Unmarshaler)(nil)).Elem()) {
				resultType := reflect.New(b)
				result := resultType.MethodByName("UnmarshalGQL").Call([]reflect.Value{reflect.ValueOf(v)})
				err, _ := result[0].Interface().(error)
				return resultType.Elem().Interface(), err
			}

			return v, nil
		},
	})

	if err != nil {
		return err
	}

	return dec.Decode(changes)
}
