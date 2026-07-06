<template>
    <f7-page ptr @ptr:refresh="reload">
        <f7-navbar :title="tt('Budget Report')" :back-link="tt('Back')">
            <f7-nav-right>
                <f7-link href="/budgets">{{ tt('Manage') }}</f7-link>
            </f7-nav-right>
        </f7-navbar>

        <f7-block class="display-flex align-items-center justify-content-space-between no-margin-bottom">
            <f7-link @click="shiftMonth(-1)">{{ tt('Previous') }}</f7-link>
            <strong>{{ selectedYearMonthText }}</strong>
            <f7-link @click="shiftMonth(1)">{{ tt('Next') }}</f7-link>
        </f7-block>

        <f7-list strong inset dividers class="margin-top" v-if="loading">
            <f7-list-item :key="itemIdx" title="Budget" after="0%" v-for="itemIdx in [1, 2, 3]"></f7-list-item>
        </f7-list>

        <f7-block strong inset class="margin-top" v-else-if="!activeBudgetReports.length">
            {{ tt('No available budget') }}
        </f7-block>

        <div class="budget-report-list" v-else>
            <f7-card :key="item.budget.id" v-for="item in activeBudgetReports">
                <f7-card-content>
                    <div class="display-flex align-items-center">
                        <span class="budget-color-dot" :style="{ backgroundColor: item.color }"></span>
                        <strong>{{ item.name }}</strong>
                    </div>
                    <div class="budget-amount-line margin-top-half">
                        {{ formatAmountToLocalizedNumeralsWithCurrency(item.spent, defaultCurrency) }} / {{ formatAmountToLocalizedNumeralsWithCurrency(item.budget.amount, defaultCurrency) }}
                    </div>
                    <div class="budget-progress margin-top-half">
                        <div class="budget-progress-bar" :style="{ width: Math.min(item.percent, 100) + '%', backgroundColor: item.color }"></div>
                    </div>
                    <div class="display-flex justify-content-space-between margin-top-half text-color-gray">
                        <small>{{ formatPercent(item.percent) }}</small>
                        <small :class="item.remaining < 0 ? 'text-color-red' : ''">{{ getRemainingText(item.remaining) }}</small>
                    </div>
                </f7-card-content>
            </f7-card>
        </div>
    </f7-page>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue';
import type { Router } from 'framework7/types';

import { useI18n } from '@/locales/helpers.ts';
import { useI18nUIComponents } from '@/lib/ui/mobile.ts';

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

defineProps<{
    f7router: Router.Router;
}>();

interface BudgetReportItem {
    budget: Budget;
    name: string;
    color: string;
    spent: number;
    percent: number;
    remaining: number;
}

const { tt, formatAmountToLocalizedNumeralsWithCurrency } = useI18n();
const { showToast } = useI18nUIComponents();
const accountsStore = useAccountsStore();
const budgetsStore = useBudgetsStore();
const categoriesStore = useTransactionCategoriesStore();
const exchangeRatesStore = useExchangeRatesStore();
const settingsStore = useSettingsStore();
const userStore = useUserStore();

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
        return 'var(--f7-theme-color)';
    }

    const category = categoriesStore.allTransactionCategoriesMap[budget.categoryId];
    const parentCategory = category && category.parentId !== '0' ? categoriesStore.allTransactionCategoriesMap[category.parentId] : category;
    return parentCategory ? `#${parentCategory.color}` : 'var(--f7-theme-color)';
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

function reload(done?: () => void): void {
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
        done?.();
    }).catch(error => {
        loading.value = false;
        done?.();
        if (!error.processed) {
            showToast(error.message || error);
        }
    });
}

reload();
</script>

<style scoped>
.budget-report-list {
    padding-bottom: var(--f7-card-margin-vertical);
}

.budget-color-dot {
    display: inline-block;
    width: 0.75rem;
    height: 0.75rem;
    border-radius: 999px;
    margin-inline-end: 0.5rem;
}

.budget-amount-line {
    font-weight: 600;
}

.budget-progress {
    height: 0.75rem;
    border-radius: 999px;
    background: rgba(128, 128, 128, 0.2);
    overflow: hidden;
}

.budget-progress-bar {
    height: 100%;
    border-radius: 999px;
}
</style>
