import { ref } from 'vue';
import { defineStore } from 'pinia';

import { itemAndIndex } from '@/core/base.ts';

import {
    type BudgetInfoResponse,
    type BudgetNewDisplayOrderRequest,
    Budget
} from '@/models/budget.ts';

import { isEquals } from '@/lib/common.ts';
import services, { type ApiResponsePromise } from '@/lib/services.ts';
import logger from '@/lib/logger.ts';

export const useBudgetsStore = defineStore('budgets', () => {
    const allBudgets = ref<Budget[]>([]);
    const allBudgetsMap = ref<Record<string, Budget>>({});
    const budgetListStateInvalid = ref<boolean>(true);

    function loadBudgetList(budgets: Budget[]): void {
        allBudgets.value = budgets;
        allBudgetsMap.value = {};

        for (const budget of budgets) {
            allBudgetsMap.value[budget.id] = budget;
        }
    }

    function addBudgetToBudgetList(budget: Budget): void {
        allBudgets.value.push(budget);
        allBudgetsMap.value[budget.id] = budget;
    }

    function updateBudgetInBudgetList(currentBudget: Budget): void {
        for (const [budget, index] of itemAndIndex(allBudgets.value)) {
            if (budget.id === currentBudget.id) {
                allBudgets.value.splice(index, 1, currentBudget);
                break;
            }
        }

        allBudgetsMap.value[currentBudget.id] = currentBudget;
    }

    function removeBudgetFromBudgetList(currentBudget: Budget): void {
        for (const [budget, index] of itemAndIndex(allBudgets.value)) {
            if (budget.id === currentBudget.id) {
                allBudgets.value.splice(index, 1);
                break;
            }
        }

        if (allBudgetsMap.value[currentBudget.id]) {
            delete allBudgetsMap.value[currentBudget.id];
        }
    }

    function updateBudgetListInvalidState(invalidState: boolean): void {
        budgetListStateInvalid.value = invalidState;
    }

    function resetBudgets(): void {
        allBudgets.value = [];
        allBudgetsMap.value = {};
        budgetListStateInvalid.value = true;
    }

    function loadAllBudgets({ force }: { force?: boolean }): Promise<Budget[]> {
        if (!force && !budgetListStateInvalid.value) {
            return Promise.resolve(allBudgets.value);
        }

        return new Promise((resolve, reject) => {
            services.getAllBudgets().then(response => {
                const data = response.data;

                if (!data || !data.success || !data.result) {
                    reject({ message: 'Unable to retrieve budget list' });
                    return;
                }

                if (budgetListStateInvalid.value) {
                    updateBudgetListInvalidState(false);
                }

                const budgets = Budget.ofMulti(data.result);

                if (force && data.result && isEquals(allBudgets.value, budgets)) {
                    reject({ message: 'Budget list is up to date', isUpToDate: true });
                    return;
                }

                loadBudgetList(budgets);
                resolve(budgets);
            }).catch(error => {
                logger.error(force ? 'failed to force load budget list' : 'failed to load budget list', error);

                if (error.response && error.response.data && error.response.data.errorMessage) {
                    reject({ error: error.response.data });
                } else if (!error.processed) {
                    reject({ message: 'Unable to retrieve budget list' });
                } else {
                    reject(error);
                }
            });
        });
    }

    function saveBudget({ budget, isEdit, clientSessionId }: { budget: Budget, isEdit: boolean, clientSessionId: string }): Promise<Budget> {
        return new Promise((resolve, reject) => {
            let promise: ApiResponsePromise<BudgetInfoResponse>;

            if (!isEdit) {
                promise = services.addBudget(budget.toCreateRequest(clientSessionId));
            } else {
                promise = services.modifyBudget(budget.toModifyRequest());
            }

            promise.then(response => {
                const data = response.data;

                if (!data || !data.success || !data.result) {
                    reject({ message: isEdit ? 'Unable to save budget' : 'Unable to add budget' });
                    return;
                }

                const savedBudget = Budget.of(data.result);

                if (isEdit) {
                    updateBudgetInBudgetList(savedBudget);
                } else {
                    addBudgetToBudgetList(savedBudget);
                }

                resolve(savedBudget);
            }).catch(error => {
                logger.error('failed to save budget', error);

                if (error.response && error.response.data && error.response.data.errorMessage) {
                    reject({ error: error.response.data });
                } else if (!error.processed) {
                    reject({ message: isEdit ? 'Unable to save budget' : 'Unable to add budget' });
                } else {
                    reject(error);
                }
            });
        });
    }

    function updateBudgetDisplayOrders(): Promise<boolean> {
        const newDisplayOrders: BudgetNewDisplayOrderRequest[] = [];

        for (const [budget, index] of itemAndIndex(allBudgets.value)) {
            newDisplayOrders.push({
                id: budget.id,
                displayOrder: index + 1
            });
        }

        return new Promise((resolve, reject) => {
            services.moveBudget({ newDisplayOrders }).then(response => {
                const data = response.data;

                if (!data || !data.success || !data.result) {
                    reject({ message: 'Unable to move budget' });
                    return;
                }

                updateBudgetListInvalidState(true);
                resolve(data.result);
            }).catch(error => {
                logger.error('failed to save budget display order', error);

                if (error.response && error.response.data && error.response.data.errorMessage) {
                    reject({ error: error.response.data });
                } else if (!error.processed) {
                    reject({ message: 'Unable to move budget' });
                } else {
                    reject(error);
                }
            });
        });
    }

    function deleteBudget({ budget }: { budget: Budget }): Promise<boolean> {
        return new Promise((resolve, reject) => {
            services.deleteBudget({ id: budget.id }).then(response => {
                const data = response.data;

                if (!data || !data.success || !data.result) {
                    reject({ message: 'Unable to delete this budget' });
                    return;
                }

                removeBudgetFromBudgetList(budget);
                resolve(data.result);
            }).catch(error => {
                logger.error('failed to delete budget', error);

                if (error.response && error.response.data && error.response.data.errorMessage) {
                    reject({ error: error.response.data });
                } else if (!error.processed) {
                    reject({ message: 'Unable to delete this budget' });
                } else {
                    reject(error);
                }
            });
        });
    }

    return {
        allBudgets,
        allBudgetsMap,
        budgetListStateInvalid,
        updateBudgetListInvalidState,
        resetBudgets,
        loadAllBudgets,
        saveBudget,
        updateBudgetDisplayOrders,
        deleteBudget
    };
});
