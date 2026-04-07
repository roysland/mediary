package server

import (
	"database/sql"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"roysland.me/symptomstracker/internal/db"
	"roysland.me/symptomstracker/internal/i18n"
)

type onboardingStep struct {
	Number      int
	TemplateKey string
	TitleKey    string
	Image       string
	ImageDesc   string
}

var onboardingSteps = []onboardingStep{
	{
		Number:      1,
		TemplateKey: "onboarding_step1_passkey",
		TitleKey:    "onboarding.step1.title",
		Image:       "onboard_passkey.png",
		ImageDesc:   "A smartphone with a fingerprint sensor glowing, next to a broken padlock representing a discarded password, on a calm blue background.",
	},
	{
		Number:      2,
		TemplateKey: "onboarding_step2_language",
		TitleKey:    "onboarding.step2.title",
		Image:       "onboard_language.png",
		ImageDesc:   "A globe with speech bubbles in different languages floating around it, soft pastel colors.",
	},
	{
		Number:      3,
		TemplateKey: "onboarding_step3_trackables",
		TitleKey:    "onboarding.step3.title",
		Image:       "onboard_symptoms.png",
		ImageDesc:   "A simple checklist with colorful icons for fatigue, sleep, and mood, being ticked off one by one.",
	},
	{
		Number:      4,
		TemplateKey: "onboarding_step4_audio",
		TitleKey:    "onboarding.step4.title",
		Image:       "onboard_mic.png",
		ImageDesc:   "A microphone with sound waves, and a small document appearing beside it to represent transcription.",
	},
	{
		Number:      5,
		TemplateKey: "onboarding_step5_navigation",
		TitleKey:    "onboarding.step5.title",
		Image:       "onboard_navigation.png",
		ImageDesc:   "Two paths diverging, one leading to a list view and one leading to a diary entry form, with clear signpost labels.",
	},
}

type onboardingViewData struct {
	Step             onboardingStep
	Steps            []onboardingStep
	CurrentStep      int
	NextStep         int
	IsFinalStep      bool
	Language         string
	TrackablePresets []db.TrackableTemplate
	IsPreview        bool
}

func onboardingStepByNumber(stepNumber int) (onboardingStep, bool) {
	for _, step := range onboardingSteps {
		if step.Number == stepNumber {
			return step, true
		}
	}

	return onboardingStep{}, false
}

func (s *Server) onboardingRoot(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodGet) {
		return
	}

	http.Redirect(w, r, "/onboarding/1", http.StatusSeeOther)
}

func (s *Server) onboardingStepRoute(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		s.onboardingStep(w, r)
		return
	}

	if r.Method == http.MethodPost {
		s.onboardingStepPost(w, r)
		return
	}

	w.Header().Set("Allow", http.MethodGet+", "+http.MethodPost)
	respondMethodNotAllowed(w, r)
}

func (s *Server) onboardingPreview(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodGet) {
		return
	}

	step, _ := onboardingStepByNumber(1)
	s.renderPage(w, r, "onboarding_title", "onboarding_content", onboardingViewData{
		Step:        step,
		Steps:       onboardingSteps,
		CurrentStep: 1,
		NextStep:    2,
		IsPreview:   true,
		Language:    i18n.DefaultLocale,
	})
}

func (s *Server) onboardingStep(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodGet) {
		return
	}

	userID, ok := requireUserID(w, r)
	if !ok {
		return
	}

	isComplete, err := s.isOnboardingComplete(r.Context(), userID)
	if err != nil {
		log.Printf("failed to read onboarding setting for user %d: %v", userID, err)
	}
	if err == nil && isComplete {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	stepNumberRaw := r.PathValue("step")
	stepNumber, err := strconv.Atoi(stepNumberRaw)
	if err != nil {
		respondNotFound(w, r, "Not found")
		return
	}

	step, exists := onboardingStepByNumber(stepNumber)
	if !exists {
		respondNotFound(w, r, "Not found")
		return
	}

	viewData := onboardingViewData{
		Step:        step,
		Steps:       onboardingSteps,
		CurrentStep: step.Number,
		NextStep:    nextOnboardingStep(step.Number),
		IsFinalStep: step.Number == len(onboardingSteps),
	}

	settings, loadErr := s.loadUserSettings(r.Context(), userID)
	if loadErr == nil {
		viewData.Language = settings.Language
	} else {
		viewData.Language = i18n.DefaultLocale
	}

	if step.Number == 3 {
		trackableTemplates, presetErr := s.queries.GetTrackableTemplates(r.Context(), userID)
		if presetErr != nil {
			respondInternalError(w, r, "Failed to fetch trackable presets")
			return
		}

		locale := viewData.Language
		if !i18n.IsSupportedLocale(locale) {
			locale = i18n.DefaultLocale
		}
		viewData.TrackablePresets = localizeTrackableTemplates(locale, trackableTemplates)
	}

	s.renderPage(w, r, "onboarding_title", "onboarding_content", viewData)
}

func (s *Server) onboardingStepPost(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodPost) {
		return
	}

	userID, ok := requireUserID(w, r)
	if !ok {
		return
	}

	stepNumberRaw := r.PathValue("step")
	stepNumber, err := strconv.Atoi(stepNumberRaw)
	if err != nil {
		respondNotFound(w, r, "Not found")
		return
	}

	step, exists := onboardingStepByNumber(stepNumber)
	if !exists {
		respondNotFound(w, r, "Not found")
		return
	}

	if !requireParsedForm(w, r) {
		return
	}

	isSkip := strings.HasSuffix(r.URL.Path, "/skip")
	nowUTC := time.Now().UTC().Unix()

	if step.Number == 2 && !isSkip {
		language, valueErr := requireOneOf(r.FormValue("language"), "language", i18n.LocaleEnglish, i18n.LocaleNorwegian)
		if valueErr != nil {
			respondBadRequest(w, r, valueErr.Error())
			return
		}

		if err := s.queries.UpsertSetting(r.Context(), db.UpsertSettingParams{
			UserID:      userID,
			SettingsKey: "language",
			SettingsValue: sql.NullString{
				String: language,
				Valid:  true,
			},
			CreatedAtUtc: nowUTC,
		}); err != nil {
			respondInternalError(w, r, "Failed to save language")
			return
		}
	}

	if step.Number == len(onboardingSteps) {
		if err := s.queries.UpsertSetting(r.Context(), db.UpsertSettingParams{
			UserID:      userID,
			SettingsKey: "onboarding_complete",
			SettingsValue: sql.NullString{
				String: "1",
				Valid:  true,
			},
			CreatedAtUtc: nowUTC,
		}); err != nil {
			log.Printf("failed to persist onboarding completion for user %d: %v", userID, err)
		}

		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	http.Redirect(w, r, "/onboarding/"+strconv.Itoa(nextOnboardingStep(step.Number)), http.StatusSeeOther)
}

func nextOnboardingStep(currentStep int) int {
	if currentStep >= len(onboardingSteps) {
		return len(onboardingSteps)
	}

	return currentStep + 1
}
