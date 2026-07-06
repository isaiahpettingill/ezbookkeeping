export enum BudgetType {
    TotalExpense = 1,
    PrimaryCategory = 2,
    SecondaryCategory = 3
}

export class Budget implements BudgetInfoResponse {
    public id: string;
    public type: BudgetType;
    public categoryId: string;
    public amount: number;
    public startYearMonth: number;
    public endYearMonth: number;
    public displayOrder: number;

    private constructor(id: string, type: BudgetType, categoryId: string, amount: number, startYearMonth: number, endYearMonth: number, displayOrder: number) {
        this.id = id;
        this.type = type;
        this.categoryId = categoryId;
        this.amount = amount;
        this.startYearMonth = startYearMonth;
        this.endYearMonth = endYearMonth;
        this.displayOrder = displayOrder;
    }

    public toCreateRequest(clientSessionId: string): BudgetCreateRequest {
        return {
            type: this.type,
            categoryId: this.categoryId,
            amount: this.amount,
            startYearMonth: this.startYearMonth,
            endYearMonth: this.endYearMonth,
            clientSessionId: clientSessionId
        };
    }

    public toModifyRequest(): BudgetModifyRequest {
        return {
            id: this.id,
            type: this.type,
            categoryId: this.categoryId,
            amount: this.amount,
            startYearMonth: this.startYearMonth,
            endYearMonth: this.endYearMonth
        };
    }

    public static of(budgetResponse: BudgetInfoResponse): Budget {
        return new Budget(
            budgetResponse.id,
            budgetResponse.type,
            budgetResponse.categoryId,
            budgetResponse.amount,
            budgetResponse.startYearMonth,
            budgetResponse.endYearMonth,
            budgetResponse.displayOrder
        );
    }

    public static ofMulti(budgetResponses: BudgetInfoResponse[]): Budget[] {
        const budgets: Budget[] = [];

        for (const budgetResponse of budgetResponses) {
            budgets.push(Budget.of(budgetResponse));
        }

        return budgets;
    }

    public static createNewBudget(type?: BudgetType, categoryId?: string): Budget {
        return new Budget('', type || BudgetType.TotalExpense, categoryId || '0', 0, 0, 0, 0);
    }
}

export interface BudgetCreateRequest {
    readonly type: BudgetType;
    readonly categoryId: string;
    readonly amount: number;
    readonly startYearMonth: number;
    readonly endYearMonth: number;
    readonly clientSessionId: string;
}

export interface BudgetModifyRequest {
    readonly id: string;
    readonly type: BudgetType;
    readonly categoryId: string;
    readonly amount: number;
    readonly startYearMonth: number;
    readonly endYearMonth: number;
}

export interface BudgetMoveRequest {
    readonly newDisplayOrders: BudgetNewDisplayOrderRequest[];
}

export interface BudgetNewDisplayOrderRequest {
    readonly id: string;
    readonly displayOrder: number;
}

export interface BudgetDeleteRequest {
    readonly id: string;
}

export interface BudgetInfoResponse {
    readonly id: string;
    readonly type: BudgetType;
    readonly categoryId: string;
    readonly amount: number;
    readonly startYearMonth: number;
    readonly endYearMonth: number;
    readonly displayOrder: number;
}
