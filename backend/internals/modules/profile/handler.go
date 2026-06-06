package profile

import (
    "net/http"

    "github.com/gin-gonic/gin"
    "student_portal/backend/internals/middleware"
    "student_portal/backend/internals/utils"
)

type ProfileHandler struct {
    profileService *ProfileService
}

func NewProfileHandler(service *ProfileService) *ProfileHandler {
    return &ProfileHandler{profileService: service}
}

type createProfileRequest struct {
    FullName   string `json:"full_name"`
    RollNumber string `json:"roll_number"`
    Year       int    `json:"year"`
    Branch     string `json:"branch"`
    Phone      string `json:"phone"`
    AvatarURL  string `json:"avatar_url"`
    Bio        string `json:"bio"`
}

type updateProfileRequest struct {
    FullName  *string `json:"full_name"`
    Year      *int    `json:"year"`
    Branch    *string `json:"branch"`
    Phone     *string `json:"phone"`
    AvatarURL *string `json:"avatar_url"`
    Bio       *string `json:"bio"`
}

func (h *ProfileHandler) GetMyProfile(c *gin.Context) {
    user, ok := middleware.UserFromContext(c.Request.Context())
    if !ok {
        utils.SendUnauthorized(c.Writer)
        return
    }

    profile, err := h.profileService.GetProfileByUserID(user.ID)
    if err == ErrProfileNotFound {
        utils.SendError(c.Writer, http.StatusNotFound, "profile not found", "PROFILE_NOT_FOUND")
        return
    }
    if err != nil {
        utils.SendInternalError(c.Writer, err, nil)
        return
    }

    utils.SendSuccess(c.Writer, http.StatusOK, "profile retrieved", profile)
}

func (h *ProfileHandler) CreateProfile(c *gin.Context) {
    user, ok := middleware.UserFromContext(c.Request.Context())
    if !ok {
        utils.SendUnauthorized(c.Writer)
        return
    }

    var req createProfileRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        utils.SendValidationError(c.Writer, "invalid request body", nil)
        return
    }

    if req.FullName == "" || req.RollNumber == "" || req.Year == 0 || req.Branch == "" {
        utils.SendValidationError(c.Writer, "missing required fields", nil)
        return
    }
    if req.Year < 1 || req.Year > 5 {
        utils.SendValidationError(c.Writer, "invalid year", nil)
        return
    }

    profile, err := h.profileService.CreateProfile(user.ID, &CreateProfileData{
        FullName:   req.FullName,
        RollNumber: req.RollNumber,
        Year:       req.Year,
        Branch:     req.Branch,
        Phone:      req.Phone,
        AvatarURL:  req.AvatarURL,
        Bio:        req.Bio,
    })
    if err == ErrProfileAlreadyExists {
        utils.SendError(c.Writer, http.StatusConflict, "profile already exists", "PROFILE_EXISTS")
        return
    }
    if err == ErrRollNumberTaken {
        utils.SendError(c.Writer, http.StatusConflict, "roll number already taken", "ROLL_NUMBER_TAKEN")
        return
    }
    if err != nil {
        utils.SendInternalError(c.Writer, err, nil)
        return
    }

    utils.SendSuccess(c.Writer, http.StatusCreated, "profile created", profile)
}

func (h *ProfileHandler) UpdateProfile(c *gin.Context) {
    user, ok := middleware.UserFromContext(c.Request.Context())
    if !ok {
        utils.SendUnauthorized(c.Writer)
        return
    }

    var req updateProfileRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        utils.SendValidationError(c.Writer, "invalid request body", nil)
        return
    }
    if req.Year != nil && (*req.Year < 1 || *req.Year > 5) {
        utils.SendValidationError(c.Writer, "invalid year", nil)
        return
    }

    profile, err := h.profileService.UpdateProfile(user.ID, &UpdateProfileData{
        FullName:  req.FullName,
        Year:      req.Year,
        Branch:    req.Branch,
        Phone:     req.Phone,
        AvatarURL: req.AvatarURL,
        Bio:       req.Bio,
    })
    if err == ErrProfileNotFound {
        utils.SendError(c.Writer, http.StatusNotFound, "profile not found", "PROFILE_NOT_FOUND")
        return
    }
    if err != nil {
        utils.SendInternalError(c.Writer, err, nil)
        return
    }

    utils.SendSuccess(c.Writer, http.StatusOK, "profile updated", profile)
}

func (h *ProfileHandler) GetProfileByID(c *gin.Context) {
    user, ok := middleware.UserFromContext(c.Request.Context())
    if !ok {
        utils.SendUnauthorized(c.Writer)
        return
    }

    userID := c.Param("userID")
    if userID == "" {
        utils.SendValidationError(c.Writer, "missing userID", nil)
        return
    }

    profile, err := h.profileService.GetProfileByID(userID, user.Role, user.CouncilCodes)
    if err == ErrProfileNotFound {
        utils.SendError(c.Writer, http.StatusNotFound, "profile not found", "PROFILE_NOT_FOUND")
        return
    }
    if err != nil {
        utils.SendInternalError(c.Writer, err, nil)
        return
    }

    utils.SendSuccess(c.Writer, http.StatusOK, "profile retrieved", profile)
}
