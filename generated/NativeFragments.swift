// Generated from Swift GraphQL declarations.


typealias IosUserPageQueryData = App_V1_GetIosUserPageResponse

typealias IosUserSummaryQueryData = App_V1_GetIosUserSummaryResponse

struct IosAvatarFragmentData: Equatable {
    let avatarUrl: String
}

struct IosOrganizationBadgeFragmentData: Equatable {
    let name: String
    let website: String
}

struct IosUserCardFragmentData_Organization: Equatable {
    let name: String
    let website: String
}

struct IosUserCardFragmentData: Equatable {
    let avatarUrl: String
    let id: String
    let name: String
    let organization: IosUserCardFragmentData_Organization?
}

extension IosUserCardFragmentData {
    func asIosAvatarFragmentData() -> IosAvatarFragmentData {
        IosAvatarFragmentData(
            avatarUrl: self.avatarUrl
        )
    }
}

extension App_V1_GetIosUserPageResponse_User {
    func asIosAvatarFragmentData() -> IosAvatarFragmentData {
        IosAvatarFragmentData(
            avatarUrl: self.avatarURL
        )
    }
}

extension IosUserCardFragmentData_Organization {
    func asIosOrganizationBadgeFragmentData() -> IosOrganizationBadgeFragmentData {
        IosOrganizationBadgeFragmentData(
            name: self.name,
            website: self.website
        )
    }
}

extension App_V1_GetIosUserPageResponse_User_Organization {
    func asIosOrganizationBadgeFragmentData() -> IosOrganizationBadgeFragmentData {
        IosOrganizationBadgeFragmentData(
            name: self.name,
            website: self.website
        )
    }
}

extension IosOrganizationBadgeFragmentData {
    func asIosUserCardFragmentData_Organization() -> IosUserCardFragmentData_Organization {
        IosUserCardFragmentData_Organization(
            name: self.name,
            website: self.website
        )
    }
}

extension App_V1_GetIosUserPageResponse_User_Organization {
    func asIosUserCardFragmentData_Organization() -> IosUserCardFragmentData_Organization {
        IosUserCardFragmentData_Organization(
            name: self.name,
            website: self.website
        )
    }
}

extension App_V1_GetIosUserPageResponse_User {
    func asIosUserCardFragmentData() -> IosUserCardFragmentData {
        IosUserCardFragmentData(
            avatarUrl: self.avatarURL,
            id: self.id,
            name: self.name,
            organization: self.hasOrganization ? self.organization.asIosUserCardFragmentData_Organization() : nil
        )
    }
}
