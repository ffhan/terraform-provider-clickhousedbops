package rowpolicy

import (
	"context"
	_ "embed"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/ClickHouse/terraform-provider-clickhousedbops/internal/dbops"
)

//go:embed rowpolicy.md
var rowPolicyDescription string

var (
	_ resource.Resource              = &Resource{}
	_ resource.ResourceWithConfigure = &Resource{}
)

func NewResource() resource.Resource {
	return &Resource{}
}

type Resource struct {
	client dbops.Client
}

func (r *Resource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_row_policy"
}

func (r *Resource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"cluster_name": schema.StringAttribute{
				Optional:    true,
				Description: "Name of the cluster to create the resource into. If omitted, resource will be created on the replica hit by the query.\nThis field must be left null when using a ClickHouse Cloud cluster.\n",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "The name of the row policy.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"database_name": schema.StringAttribute{
				Required:    true,
				Description: "The database of the table to apply the row policy to.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"table_name": schema.StringAttribute{
				Required:    true,
				Description: "The table to apply the row policy to.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"select_filter": schema.StringAttribute{
				Required:    true,
				Description: "The filter expression used in the USING clause. For example: `tenant_id = 'abc'`.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"is_restrictive": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
				Description: "If true, the policy is restrictive (AND logic). If false (default), the policy is permissive (OR logic).",
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.RequiresReplace(),
				},
			},
			"grantee_user_name": schema.StringAttribute{
				Optional:    true,
				Description: "Name of the user to apply the row policy to.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.ConflictsWith(path.Expressions{path.MatchRoot("grantee_role_name")}...),
					stringvalidator.AtLeastOneOf(path.Expressions{
						path.MatchRoot("grantee_user_name"),
						path.MatchRoot("grantee_role_name"),
					}...),
				},
			},
			"grantee_role_name": schema.StringAttribute{
				Optional:    true,
				Description: "Name of the role to apply the row policy to.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.ConflictsWith(path.Expressions{path.MatchRoot("grantee_user_name")}...),
					stringvalidator.AtLeastOneOf(path.Expressions{
						path.MatchRoot("grantee_user_name"),
						path.MatchRoot("grantee_role_name"),
					}...),
				},
			},
		},
		MarkdownDescription: rowPolicyDescription,
	}
}

func (r *Resource) Configure(_ context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	r.client = req.ProviderData.(dbops.Client)
}

func (r *Resource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if req.Plan.Raw.IsNull() {
		return
	}

	var config RowPolicy
	diags := req.Config.Get(ctx, &config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if r.client != nil {
		isReplicatedStorage, err := r.client.IsReplicatedStorage(ctx)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error Checking if service is using replicated storage",
				fmt.Sprintf("%+v\n", err),
			)
			return
		}

		if isReplicatedStorage && !config.ClusterName.IsNull() {
			resp.Diagnostics.AddWarning(
				"Invalid configuration",
				"Your ClickHouse cluster is using Replicated storage for grants, please remove the 'cluster_name' attribute from your RowPolicy resource definition if you encounter any errors.",
			)
		}
	}
}

func (r *Resource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan RowPolicy
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	rp := dbops.RowPolicy{
		Name:            plan.Name.ValueString(),
		Database:        plan.Database.ValueString(),
		Table:           plan.Table.ValueString(),
		SelectFilter:    plan.SelectFilter.ValueString(),
		IsRestrictive:   plan.IsRestrictive.ValueBool(),
		GranteeUserName: plan.GranteeUserName.ValueStringPointer(),
		GranteeRoleName: plan.GranteeRoleName.ValueStringPointer(),
	}

	created, err := r.client.CreateRowPolicy(ctx, rp, plan.ClusterName.ValueStringPointer())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Creating ClickHouse Row Policy",
			"Could not create row policy, unexpected error: "+err.Error(),
		)
		return
	}

	if created == nil {
		resp.Diagnostics.AddError(
			"Error Creating ClickHouse Row Policy",
			"The row policy was created but could not be found in system.row_policies.",
		)
		return
	}

	state := RowPolicy{
		ClusterName:     plan.ClusterName,
		Name:            types.StringValue(created.Name),
		Database:        types.StringValue(created.Database),
		Table:           types.StringValue(created.Table),
		SelectFilter:    types.StringValue(created.SelectFilter),
		IsRestrictive:   types.BoolValue(created.IsRestrictive),
		GranteeUserName: types.StringPointerValue(created.GranteeUserName),
		GranteeRoleName: types.StringPointerValue(created.GranteeRoleName),
	}

	diags = resp.State.Set(ctx, state)
	resp.Diagnostics.Append(diags...)
}

func (r *Resource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state RowPolicy
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	rp := dbops.RowPolicy{
		Name:            state.Name.ValueString(),
		Database:        state.Database.ValueString(),
		Table:           state.Table.ValueString(),
		SelectFilter:    state.SelectFilter.ValueString(),
		IsRestrictive:   state.IsRestrictive.ValueBool(),
		GranteeUserName: state.GranteeUserName.ValueStringPointer(),
		GranteeRoleName: state.GranteeRoleName.ValueStringPointer(),
	}

	result, err := r.client.GetRowPolicy(ctx, &rp, state.ClusterName.ValueStringPointer())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading ClickHouse Row Policy",
			"Could not read row policy, unexpected error: "+err.Error(),
		)
		return
	}

	if result != nil {
		state.Name = types.StringValue(result.Name)
		state.Database = types.StringValue(result.Database)
		state.Table = types.StringValue(result.Table)
		state.SelectFilter = types.StringValue(result.SelectFilter)
		state.IsRestrictive = types.BoolValue(result.IsRestrictive)
		state.GranteeUserName = types.StringPointerValue(result.GranteeUserName)
		state.GranteeRoleName = types.StringPointerValue(result.GranteeRoleName)

		diags = resp.State.Set(ctx, &state)
		resp.Diagnostics.Append(diags...)
	} else {
		resp.State.RemoveResource(ctx)
	}
}

func (r *Resource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state RowPolicy
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	diags = req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	rp := dbops.RowPolicy{
		Name:            plan.Name.ValueString(),
		Database:        plan.Database.ValueString(),
		Table:           plan.Table.ValueString(),
		SelectFilter:    plan.SelectFilter.ValueString(),
		IsRestrictive:   plan.IsRestrictive.ValueBool(),
		GranteeUserName: plan.GranteeUserName.ValueStringPointer(),
		GranteeRoleName: plan.GranteeRoleName.ValueStringPointer(),
	}

	updated, err := r.client.UpdateRowPolicy(ctx, rp, plan.ClusterName.ValueStringPointer())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Updating ClickHouse Row Policy",
			"Could not update row policy, unexpected error: "+err.Error(),
		)
		return
	}

	if updated != nil {
		state.Name = types.StringValue(updated.Name)
		state.Database = types.StringValue(updated.Database)
		state.Table = types.StringValue(updated.Table)
		state.SelectFilter = types.StringValue(updated.SelectFilter)
		state.IsRestrictive = types.BoolValue(updated.IsRestrictive)
		state.GranteeUserName = types.StringPointerValue(updated.GranteeUserName)
		state.GranteeRoleName = types.StringPointerValue(updated.GranteeRoleName)

		diags = resp.State.Set(ctx, &state)
		resp.Diagnostics.Append(diags...)
	} else {
		resp.Diagnostics.AddError(
			"Error Updating ClickHouse Row Policy",
			"The row policy was updated but could not be found in system.row_policies.",
		)
	}
}

func (r *Resource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state RowPolicy
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteRowPolicy(ctx, state.Name.ValueString(), state.Database.ValueString(), state.Table.ValueString(), state.ClusterName.ValueStringPointer())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Deleting ClickHouse Row Policy",
			"Could not delete row policy, unexpected error: "+err.Error(),
		)
		return
	}
}
