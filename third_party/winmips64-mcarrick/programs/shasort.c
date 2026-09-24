#include <stdio.h>

int array[]={2,3,7,5,1,9,6,9,3,2,4,8,3,6,3,8,9,0,1,7};


void InsertionSort(int a[],int len)
{
    int i,j,B;
	for (i = 0; i < len; i++) 
    {
	    j = i;
	    B = a[i];
	    while ((j >= 0) && (a[j-1] > B)) 
        {     
	        a[j] = a[j-1];
	        j--;	       
	    }
	    a[j] = B;
    }
}




void ShakerSort(int a[],int len)
{
    int T,j,mn,mx;
	int i = 0;
	int k = len;
	while (i < k) 
    {
        mn=i;
        mx=i;
        for (j=i;j<k;j++) 
        {
		    if (a[j] < a[mn]) mn = j;
            if (a[j] > a[mx]) mx = j;
	    }

        T = a[mn];
        a[mn] = a[i];
	    a[i] = T;

        k--;

	    if (mx==i) 
        {
	        T = a[mn];
	        a[mn] = a[k];
	 	    a[k] = T;
	    } 
        else 
        {
	        T = a[mx];
	        a[mx] = a[k];
	        a[k] = T;
	    }	
        
	    i++;
    }
}

int main()
{
    int i;
    InsertionSort(array,20);
//    ShakerSort(array,20);
    for (i=0;i<20;i++)
    {
        printf("a[%d]= %d\n",i,array[i]);
    }
    return 0;
}
