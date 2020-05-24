package statistics

import (
	"github.com/GoodCodingFriends/animekai/api"
	"github.com/GoodCodingFriends/animekai/errors"
	"github.com/morikuni/failure"
)

func validateGetDashboardRequest(r *api.GetDashboardRequest) error {
	if r.WorkPageSize == 0 {
		return failure.New(errors.InvalidArgument, failure.Message("work_page_size must be greater than 0"))
	}
	return nil
}

func validateListWorksRequest(r *api.ListWorksRequest) error {
	if r.State == api.WorkState_WORK_STATE_UNSPECIFIED {
		return failure.New(errors.InvalidArgument, failure.Message("state must be specified"))
	}
	if r.PageSize == 0 {
		return failure.New(errors.InvalidArgument, failure.Message("page_sizse must be greater than 0"))
	}
	return nil
}
