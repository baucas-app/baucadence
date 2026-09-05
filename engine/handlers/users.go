package handlers

import (
	"errors"
	"net/http"

	"github.com/gabehf/koito/engine/middleware"
	"github.com/gabehf/koito/internal/db"
	"github.com/gabehf/koito/internal/logger"
	"github.com/gabehf/koito/internal/models"
	"github.com/gabehf/koito/internal/utils"
)

func ListUsersHandler(store db.UserStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		l := logger.FromContext(ctx)

		user := middleware.GetUserFromContext(ctx)
		if user == nil {
			l.Debug().Msg("ListUsersHandler: Invalid user context")
			utils.WriteError(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		if user.Role != models.UserRoleAdmin {
			l.Debug().Msg("ListUsersHandler: Non-admin user attempted to list users")
			utils.WriteError(w, "forbidden", http.StatusForbidden)
			return
		}

		users, err := store.ListUsers(ctx)
		if err != nil {
			l.Error().Err(err).Msg("ListUsersHandler: Failed to list users")
			utils.WriteError(w, "failed to list users", http.StatusInternalServerError)
			return
		}

		l.Debug().Msgf("ListUsersHandler: Retrieved %d users", len(users))
		utils.WriteJSON(w, http.StatusOK, users)
	}
}

func CreateUserHandler(store db.UserStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		l := logger.FromContext(ctx)

		user := middleware.GetUserFromContext(ctx)
		if user == nil {
			l.Debug().Msg("CreateUserHandler: Invalid user context")
			utils.WriteError(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		if user.Role != models.UserRoleAdmin {
			l.Debug().Msg("CreateUserHandler: Non-admin user attempted to create a user")
			utils.WriteError(w, "forbidden", http.StatusForbidden)
			return
		}

		body, err := utils.DecodeBody[struct {
			Username string `json:"username"`
			Password string `json:"password"`
			Role     string `json:"role"`
		}](r)
		if err != nil || body.Username == "" || body.Password == "" {
			l.Debug().Msg("CreateUserHandler: Missing or invalid request body")
			utils.WriteError(w, "username and password required", http.StatusBadRequest)
			return
		}

		role := models.UserRole(body.Role)
		if role == "" {
			role = models.UserRoleUser
		}
		if role != models.UserRoleUser && role != models.UserRoleAdmin {
			l.Debug().Msg("CreateUserHandler: Invalid role in request body")
			utils.WriteError(w, "role must be 'user' or 'admin'", http.StatusBadRequest)
			return
		}

		created, err := store.SaveUser(ctx, db.SaveUserOpts{
			Username: body.Username,
			Password: body.Password,
			Role:     role,
		})
		if err != nil {
			if errors.Is(err, db.ErrUsernameTaken) {
				l.Debug().Msg("CreateUserHandler: Username already exists")
				utils.WriteError(w, "username already exists", http.StatusConflict)
				return
			}
			l.Error().Err(err).Msg("CreateUserHandler: Failed to save user")
			utils.WriteError(w, err.Error(), http.StatusBadRequest)
			return
		}

		l.Debug().Msgf("CreateUserHandler: Successfully created user ID %d", created.ID)
		utils.WriteJSON(w, http.StatusCreated, created)
	}
}

func DeleteUserHandler(store db.UserStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		l := logger.FromContext(ctx)

		user := middleware.GetUserFromContext(ctx)
		if user == nil {
			l.Debug().Msg("DeleteUserHandler: Invalid user context")
			utils.WriteError(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		if user.Role != models.UserRoleAdmin {
			l.Debug().Msg("DeleteUserHandler: Non-admin user attempted to delete a user")
			utils.WriteError(w, "forbidden", http.StatusForbidden)
			return
		}

		targetID, err := utils.ParseIDParam(r, "id")
		if err != nil {
			l.Debug().AnErr("error", err).Msg("DeleteUserHandler: Invalid user ID")
			utils.WriteError(w, "invalid id", http.StatusBadRequest)
			return
		}
		if targetID == user.ID {
			l.Debug().Msg("DeleteUserHandler: Admin attempted to delete their own account")
			utils.WriteError(w, "cannot delete your own account", http.StatusBadRequest)
			return
		}

		if err := store.DeleteUser(ctx, targetID); err != nil {
			if errors.Is(err, db.ErrNotFound) {
				utils.WriteError(w, "user not found", http.StatusNotFound)
				return
			}
			l.Error().Err(err).Msg("DeleteUserHandler: Failed to delete user")
			utils.WriteError(w, "failed to delete user", http.StatusInternalServerError)
			return
		}

		l.Debug().Msgf("DeleteUserHandler: Successfully deleted user ID %d", targetID)
		w.WriteHeader(http.StatusNoContent)
	}
}
