<template>
    <v-row class="match-height">
        <v-col cols="12">
            <v-card>
                <template #title>
                    <div class="title-and-toolbar d-flex align-center flex-wrap">
                        <span>{{ tt('Budget Report') }}</span>
                        <v-btn-group class="ms-4" color="default" density="comfortable" variant="outlined" divided>
                            <v-btn class="button-icon-with-direction" :icon="mdiArrowLeft" :disabled="loading" @click="shiftMonth(-1)" />
                            <v-btn :disabled="loading">{{ selectedYearMonthText }}</v-btn>
                            <v-btn class="button-icon-with-direction" :icon="mdiArrowRight" :disabled="loading" @click="shiftMonth(1)" />
                        </v-btn-group>
                        <v-btn class="ms-3" color="default" variant="outlined" to="/budgets">{{ tt('Manage Budgets') }}</v-btn>
                        <v-btn density="compact" color="default" variant="text" size="24" class="ms-2" :icon="true" :loading="loading" @click="reload">
                            <template #loader>
                                <v-progress-circular indeterminate size="20"/>
                            </template>
                            <v-icon :icon="mdiRefresh" size="24" />
                            <v-tooltip activator="parent">{{ tt('Refresh') }}</v-tooltip>
                        </v-btn>
                    </div>
                </template>

                <v-card-text>
                    <div v-if="loading">
                        <v-skeleton-loader :key="itemIdx" type="list-item-two-line" :loading="true" v-for="itemIdx in [1, 2, 3, 4]" />
                    </div>
                    <div v-else-if="!activeBudgetReports.length">
                        {{ tt('No available budget') }}
                    </div>
                    <div class="budget-report-list" v-else>
                        <div class="budget-report-item" :key="item.budget.id" v-for="item in activeBudgetReports">
                            <div class="d-flex align-center mb-2">
                                <span class="budget-color-dot" :style="{ backgroundColor: item.color }"></span>
                                <strong>{{ item.name }}</strong>
                                <v-spacer />
                                <span>{{ formatAmountToLocalizedNumeralsWithCurrency(item.spent, defaultCurrency) }} / {{ formatAmountToLocalizedNumeralsWithCurrency(item.budget.amount, defaultCurrency) }}</span>
                            </div>
                            <v-progress-linear height="14" rounded :color="item.color" :model-value="Math.min(item.percent, 100)" />
                            <div class="d-flex text-caption mt-1">
                                <span>{{ formatPercent(item.percent) }}</span>
                                <v-spacer />
                                <span :class="item.remaining < 0 ? 'text-error' : ''">{{ getRemainingText(item.remaining) }}</span>
                            </div>
                        </div>
                    </div>
                </v-card-text>
            </v-card>
        </v-col>
    </v-row>

    <snack-bar ref="snackbar" />
</template>

<script setup lang="ts">
import SnackBar from '@/components/desktop/SnackBar.vue';

import { ref, computed, useTemplateRef } from 'vue';

import { useI18n } from '@/locales/helpers.ts';
import { useAccountsStore } from '@/stores/account.ts';
import { useBudgetsStore } from '@/stores/budget.ts';
import { useExchangeRatesStore } from '@/stores/exchangeRates.ts';
import { useSettingsStore } from '@/stores/setting.ts';
import { useTransactionCategoriesStore } from '@/stores/transactionCategory.ts';
import { useUserStore } from '@/stores/user.ts';

import { CategoryType } from '@/core/category.ts';
import { TimezoneTypeForStatistics } from '@/core/timezone.ts';
import { Budget, BudgetType } from '@/models/budget.ts';
import type { TransactionStatisticResponseItem } from '@/models/transaction.ts';

import { isNumber } from '@/lib/common.ts';
import services from '@/lib/services.ts';

import { mdiArrowLeft, mdiArrowRight, mdiRefresh } from '@mdi/js';

interface BudgetReportItem {
    budget: Budget;
    name: string;
    color: string;
    spent: number;
    percent: number;
    remaining: number;
}

type SnackBarType = InstanceType<typeof SnackBar>;

const { tt, formatAmountToLocalizedNumeralsWithCurrency } = useI18n();
const accountsStore = useAccountsStore();
const budgetsStore = useBudgetsStore();
const categoriesStore = useTransactionCategoriesStore();
const exchangeRatesStore = useExchangeRatesStore();
const settingsStore = useSettingsStore();
const userStore = useUserStore();
const snackbar = useTemplateRef<SnackBarType>('snackbar');

const now = new Date();
const selectedYear = ref<number>(now.getFullYear());
const selectedMonth = ref<number>(now.getMonth() + 1);
const loading = ref<boolean>(false);
const statisticsItems = ref<TransactionStatisticResponseItem[]>([]);

const defaultCurrency = computed<string>(() => userStore.currentUserDefaultCurrency);
const selectedYearMonth = computed<number>(() => selectedYear.value * 100 + selectedMonth.value);
const selectedYearMonthText = computed<string>(() => `${selectedYear.value}-${selectedMonth.value.toString(10).padStart(2, '0')}`);

const activeBudgetReports = computed<BudgetReportItem[]>(() => {
    const reports: BudgetReportItem[] = [];

    for (const budget of budgetsStore.allBudgets) {
        if (budget.startYearMonth && selectedYearMonth.value < budget.startYearMonth) {
            continue;
        }

        if (budget.endYearMonth && selectedYearMonth.value > budget.endYearMonth) {
            continue;
        }

        const spent = getBudgetSpentAmount(budget);
        reports.push({
            budget,
            name: getBudgetName(budget),
            color: getBudgetColor(budget),
            spent,
            percent: budget.amount > 0 ? spent / budget.amount * 100 : 0,
            remaining: budget.amount - spent
        });
    }

    return reports;
});

function getBudgetSpentAmount(budget: Budget): number {
    let amount = 0;

    for (const item of statisticsItems.value) {
        const category = categoriesStore.allTransactionCategoriesMap[item.categoryId];
        const account = accountsStore.allAccountsMap[item.accountId];

        if (!category || !account || category.type !== CategoryType.Expense) {
            continue;
        }

        if (budget.type === BudgetType.PrimaryCategory) {
            const primaryCategoryId = category.parentId !== '0' ? category.parentId : category.id;

            if (primaryCategoryId !== budget.categoryId) {
                continue;
            }
        } else if (budget.type === BudgetType.SecondaryCategory && category.id !== budget.categoryId) {
            continue;
        }

        let convertedAmount = item.amount;

        if (account.currency !== defaultCurrency.value) {
            const finalAmount = exchangeRatesStore.getExchangedAmount(item.amount, account.currency, defaultCurrency.value);

            if (!isNumber(finalAmount)) {
                continue;
            }

            convertedAmount = Math.trunc(finalAmount);
        }

        amount += convertedAmount;
    }

    return amount;
}

function getBudgetName(budget: Budget): string {
    if (budget.type === BudgetType.TotalExpense) {
        return tt('Total Spend');
    }

    const category = categoriesStore.allTransactionCategoriesMap[budget.categoryId];
    const parentCategory = category && category.parentId !== '0' ? categoriesStore.allTransactionCategoriesMap[category.parentId] : undefined;
    return parentCategory ? `${parentCategory.name} / ${category?.name}` : (category?.name || tt('Unknown'));
}

function getBudgetColor(budget: Budget): string {
    if (budget.type === BudgetType.TotalExpense) {
        return 'rgb(var(--v-theme-primary))';
    }

    const category = categoriesStore.allTransactionCategoriesMap[budget.categoryId];
    const parentCategory = category && category.parentId !== '0' ? categoriesStore.allTransactionCategoriesMap[category.parentId] : category;
    return parentCategory ? `#${parentCategory.color}` : 'rgb(var(--v-theme-primary))';
}

function getRemainingText(remaining: number): string {
    if (remaining >= 0) {
        return tt('Remaining') + ': ' + formatAmountToLocalizedNumeralsWithCurrency(remaining, defaultCurrency.value);
    }

    return tt('Over Budget') + ': ' + formatAmountToLocalizedNumeralsWithCurrency(-remaining, defaultCurrency.value);
}

function formatPercent(percent: number): string {
    return `${percent.toFixed(1)}%`;
}

function getMonthRange(): { startTime: number, endTime: number } {
    const start = new Date(selectedYear.value, selectedMonth.value - 1, 1, 0, 0, 0, 0);
    const end = new Date(selectedYear.value, selectedMonth.value, 1, 0, 0, 0, 0);
    return {
        startTime: Math.floor(start.getTime() / 1000),
        endTime: Math.floor(end.getTime() / 1000) - 1
    };
}

function shiftMonth(offset: number): void {
    const date = new Date(selectedYear.value, selectedMonth.value - 1 + offset, 1);
    selectedYear.value = date.getFullYear();
    selectedMonth.value = date.getMonth() + 1;
    reload();
}

function reload(): void {
    loading.value = true;
    const range = getMonthRange();

    Promise.all([
        accountsStore.loadAllAccounts({ force: false }),
        categoriesStore.loadAllCategories({ force: false }),
        budgetsStore.loadAllBudgets({ force: false }),
        exchangeRatesStore.getLatestExchangeRates({ silent: true, force: false }),
        services.getTransactionStatistics({
            startTime: range.startTime,
            endTime: range.endTime,
            tagFilter: '',
            keyword: '',
            matchMode: 0,
            useTransactionTimezone: settingsStore.appSettings.timezoneUsedForStatisticsInHomePage === TimezoneTypeForStatistics.TransactionTimezone.type
        })
    ]).then(([, , , , statisticsResponse]) => {
        statisticsItems.value = statisticsResponse.data.result?.items || [];
        loading.value = false;
    }).catch(error => {
        loading.value = false;
        if (!error.processed) {
            snackbar.value?.showError(error);
        }
    });
}

reload();
</script>

<style scoped>
.budget-report-list {
    display: flex;
    flex-direction: column;
    gap: 1.5rem;
}

.budget-report-item {
    padding: 1rem;
    border: 1px solid rgba(var(--v-border-color), var(--v-border-opacity));
    border-radius: 0.5rem;
}

.budget-color-dot {
    width: 0.75rem;
    height: 0.75rem;
    border-radius: 999px;
    margin-inline-end: 0.5rem;
}
</style>
